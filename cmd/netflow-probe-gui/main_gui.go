//go:build darwin && cgo

// Command netflow-probe-gui is a small macOS desktop front-end (proof of
// concept) for the netflow-probe capture engine. It targets non-technical
// users: pick an interface and the Pandora collector, press Start/Stop, and —
// when macOS blocks capture — it guides the user to grant permission.
package main

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"brutus.artica.es/slerena/netflow-probe/internal/capture"
	"brutus.artica.es/slerena/netflow-probe/internal/config"
	"brutus.artica.es/slerena/netflow-probe/internal/export"
	"brutus.artica.es/slerena/netflow-probe/internal/flow"
)

//go:embed icon.png
var iconPNG []byte

const configFileName = ".netflow-probe.yaml"

type session struct {
	cancel context.CancelFunc
	src    capture.Source
	table  *flow.Table
	exp    *export.Exporter
}

type gui struct {
	win fyne.Window

	ifaceSel  *widget.Select
	collector *widget.Entry
	port      *widget.Entry
	status    *widget.Label
	logView   *widget.Entry
	startBtn  *widget.Button

	sess    *session
	running bool

	exported atomic.Uint64
	logMu    sync.Mutex
	logLines []string
}

func main() {
	a := app.New()
	a.SetIcon(fyne.NewStaticResource("icon.png", iconPNG))

	g := &gui{}
	g.win = a.NewWindow("NetFlow Probe (Pandora)")
	g.buildUI()
	g.loadConfig()
	g.win.Resize(fyne.NewSize(560, 620))
	g.win.ShowAndRun()
}

func (g *gui) buildUI() {
	g.ifaceSel = widget.NewSelect(interfaceNames(), nil)
	g.ifaceSel.PlaceHolder = "Interfaz de red"

	g.collector = widget.NewEntry()
	g.collector.PlaceHolder = "IP o host del servidor Pandora"

	g.port = widget.NewEntry()
	g.port.SetText("9995")

	g.status = widget.NewLabel("Detenido")

	g.logView = widget.NewMultiLineEntry()
	g.logView.Disable()
	g.logView.Wrapping = fyne.TextWrapWord

	g.startBtn = widget.NewButton("Iniciar", g.start)
	g.startBtn.Importance = widget.HighImportance

	permBtn := widget.NewButton("Permisos de captura…", func() { g.showPermissionDialog(nil) })
	saveBtn := widget.NewButton("Guardar configuración", func() {
		if err := g.saveConfig(); err != nil {
			g.logf("No se pudo guardar: %v", err)
		} else {
			g.logf("Configuración guardada en %s", g.configPath())
		}
	})

	form := container.NewVBox(
		widget.NewLabelWithStyle("Interfaz", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		g.ifaceSel,
		widget.NewLabelWithStyle("Colector NetFlow (Pandora)", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		g.collector,
		widget.NewLabelWithStyle("Puerto UDP", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		g.port,
		container.NewHBox(g.startBtn, permBtn, saveBtn),
		container.NewHBox(widget.NewLabelWithStyle("Estado:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), g.status),
	)

	header := container.NewHBox(
		widget.NewIcon(fyne.NewStaticResource("icon.png", iconPNG)),
		widget.NewLabelWithStyle("NetFlow Probe", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
	)

	body := container.NewBorder(
		container.NewVBox(header, widget.NewSeparator(), form, widget.NewSeparator(),
			widget.NewLabelWithStyle("Registro", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		),
		nil, nil, nil,
		g.logView,
	)
	g.win.SetContent(container.NewPadded(body))
}

// --- capture lifecycle -------------------------------------------------------

func (g *gui) start() {
	if g.running {
		return
	}

	port, err := strconv.Atoi(strings.TrimSpace(g.port.Text))
	if err != nil || port <= 0 || port > 65535 {
		g.showError("El puerto debe ser un número entre 1 y 65535.")
		return
	}
	collector := strings.TrimSpace(g.collector.Text)
	if collector == "" {
		g.showError("Indica la IP o el host del servidor Pandora (colector NetFlow).")
		return
	}
	if g.ifaceSel.Selected == "" {
		g.showError("Selecciona una interfaz de red.")
		return
	}

	cfg := config.Default()
	cfg.Interface = g.ifaceSel.Selected
	cfg.Collector = collector
	cfg.Port = port

	// Opening /dev/bpf without privileges is the usual failure point on macOS.
	src, err := capture.OpenLive(cfg.Interface, cfg.Filter)
	if err != nil {
		g.showPermissionDialog(err)
		return
	}

	exp, err := export.New(cfg.CollectorAddr(), cfg.SourceID, cfg.TemplateRefresh)
	if err != nil {
		_ = src.Close()
		g.showError(fmt.Sprintf("No se pudo preparar el exportador: %v", err))
		return
	}

	table := flow.NewTable(cfg.ActiveTimeout.Duration(), cfg.InactiveTimeout.Duration(), cfg.MaxFlows)
	ctx, cancel := context.WithCancel(context.Background())
	g.sess = &session{cancel: cancel, src: src, table: table, exp: exp}
	g.running = true
	g.exported.Store(0)
	g.setRunningUI(true)
	g.logf("Iniciado: capturando en %s → %s", cfg.Interface, cfg.CollectorAddr())

	// Reader goroutine.
	link := src.LinkType()
	go func() {
		for {
			data, _, err := src.Next()
			if err != nil {
				g.logf("Captura detenida: %v", err)
				cancel()
				return
			}
			key, l3, flags, ok := flow.Observe(data, link)
			if ok {
				table.Observe(key, l3, flags, time.Now())
			}
		}
	}()

	// Exporter goroutine owns cleanup.
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				recs := table.Flush()
				if len(recs) > 0 {
					_, _ = exp.Send(recs, time.Now())
				}
				_ = src.Close()
				_ = exp.Close()
				fyne.Do(func() {
					g.running = false
					g.sess = nil
					g.setRunningUI(false)
					g.updateStatus()
				})
				return
			case <-ticker.C:
				recs := table.Expire()
				if len(recs) == 0 {
					continue
				}
				if n, err := exp.Send(recs, time.Now()); err != nil {
					g.logf("Error al exportar: %v", err)
				} else {
					g.exported.Add(uint64(n))
					g.updateStatus()
				}
			}
		}
	}()
}

func (g *gui) stop() {
	if g.sess == nil {
		return
	}
	g.logf("Deteniendo…")
	g.sess.cancel()
}

func (g *gui) setRunningUI(running bool) {
	if running {
		g.startBtn.SetText("Detener")
	} else {
		g.startBtn.SetText("Iniciar")
	}
}

func (g *gui) updateStatus() {
	if g.running {
		fyne.Do(func() { g.status.SetText(fmt.Sprintf("Activo · %d flujos exportados", g.exported.Load())) })
		return
	}
	fyne.Do(func() { g.status.SetText(fmt.Sprintf("Detenido · %d flujos exportados", g.exported.Load())) })
}

// --- permissions (macOS) -----------------------------------------------------

// showPermissionDialog explains the macOS capture permission and offers to open
// System Settings and to grant /dev/bpf access with an administrator prompt.
func (g *gui) showPermissionDialog(cause error) {
	var msg string
	if cause != nil {
		msg = fmt.Sprintf("No se pudo iniciar la captura:\n\n%v\n\n", cause)
	}
	msg += "macOS requiere privilegios para capturar paquetes (acceso a /dev/bpf). " +
		"Puedes abrir los Ajustes del Sistema para revisar los permisos, o conceder el " +
		"acceso a /dev/bpf con tu contraseña de administrador."

	label := widget.NewLabel(msg)
	label.Wrapping = fyne.TextWrapWord

	settingsBtn := widget.NewButton("Abrir Ajustes del Sistema", func() { g.openPrivacySettings() })
	adminBtn := widget.NewButton("Conceder acceso a /dev/bpf (admin)", func() { g.grantBPFAccess() })

	content := container.NewVBox(label, layout.NewSpacer(), settingsBtn, adminBtn)
	d := dialog.NewCustom("Permisos de captura", "Cerrar", content, g.win)
	d.Resize(fyne.NewSize(480, 320))
	d.Show()
}

// openPrivacySettings opens System Settings → Privacy & Security. The URL scheme
// changed in recent macOS versions, so several are tried in order.
func (g *gui) openPrivacySettings() {
	urls := []string{
		"x-apple.systempreferences:com.apple.preference.security?Privacy_AllFiles",
		"x-apple.systempreferences:com.apple.settings.PrivacySecurity.extension?Privacy_AllFiles",
		"x-apple.systempreferences:com.apple.preference.security",
	}
	for _, u := range urls {
		if err := exec.Command("open", u).Run(); err == nil {
			g.logf("Ajustes del Sistema abiertos (Privacidad y seguridad).")
			return
		}
	}
	g.logf("No se pudieron abrir los Ajustes automáticamente: ábrelos en Privacidad y seguridad.")
}

// grantBPFAccess chmods the /dev/bpf* devices so a non-root process can open
// them. Proof of concept: this is broad, a production build should ship a
// ChmodBPF-style LaunchDaemon with a dedicated group (as Wireshark does).
func (g *gui) grantBPFAccess() {
	script := `do shell script "/bin/chmod o+rw /dev/bpf* 2>/dev/null || true" with administrator privileges`
	out, err := exec.Command("osascript", "-e", script).CombinedOutput()
	if err != nil {
		g.logf("No se pudo conceder el acceso: %v", err)
		g.showError("No se pudo conceder el acceso.\n\nPuedes ejecutarlo manualmente en Terminal:\n\nsudo chmod o+rw /dev/bpf*")
		return
	}
	if s := strings.TrimSpace(string(out)); s != "" {
		g.logf("%s", s)
	}
	g.logf("Acceso a /dev/bpf concedido. Pulsa Iniciar de nuevo.")
}

// --- helpers -----------------------------------------------------------------

func (g *gui) logf(format string, args ...any) {
	line := time.Now().Format("15:04:05") + "  " + fmt.Sprintf(format, args...)
	g.logMu.Lock()
	g.logLines = append(g.logLines, line)
	if len(g.logLines) > 500 {
		g.logLines = g.logLines[len(g.logLines)-500:]
	}
	text := strings.Join(g.logLines, "\n")
	g.logMu.Unlock()
	fyne.Do(func() { g.logView.SetText(text) })
}

func (g *gui) showError(msg string) {
	dialog.ShowError(errors.New(msg), g.win)
}

func (g *gui) configPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return configFileName
	}
	return filepath.Join(home, configFileName)
}

func (g *gui) loadConfig() {
	cfg, err := config.Load(g.configPath())
	if err != nil {
		// First run (or missing file): keep defaults and select en0 if present.
		cfg = config.Default()
	}
	if cfg.Collector != "" {
		g.collector.SetText(cfg.Collector)
	}
	if cfg.Port != 0 {
		g.port.SetText(strconv.Itoa(cfg.Port))
	}
	if cfg.Interface != "" {
		g.ifaceSel.SetSelected(cfg.Interface)
	}
	if g.ifaceSel.Selected == "" {
		for _, n := range g.ifaceSel.Options {
			if n == "en0" {
				g.ifaceSel.SetSelected("en0")
				break
			}
		}
	}
	g.updateStatus()
}

func (g *gui) saveConfig() error {
	port, err := strconv.Atoi(strings.TrimSpace(g.port.Text))
	if err != nil {
		return fmt.Errorf("puerto inválido")
	}
	cfg := config.Default()
	cfg.Interface = g.ifaceSel.Selected
	cfg.Collector = strings.TrimSpace(g.collector.Text)
	cfg.Port = port
	return cfg.Save(g.configPath())
}

// interfaceNames lists the usable network interfaces (filtering loopback and
// down interfaces).
func interfaceNames() []string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	var names []string
	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}
		names = append(names, iface.Name)
	}
	return names
}
