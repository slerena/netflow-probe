//go:build darwin && cgo

// Command netflow-probe-gui is a small macOS desktop front-end (proof of
// concept) for the netflow-probe capture engine. It targets non-technical
// users: pick an interface and the Pandora collector, press Start/Stop, and —
// when macOS blocks capture — it guides the user to grant permission.
package main

import (
	"context"
	_ "embed"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"brutus.artica.es/slerena/netflow-probe/internal/capture"
	"brutus.artica.es/slerena/netflow-probe/internal/config"
	"brutus.artica.es/slerena/netflow-probe/internal/export"
	"brutus.artica.es/slerena/netflow-probe/internal/flow"
)

//go:embed icon.png
var iconPNG []byte

//go:embed tray.png
var trayPNG []byte

const configFileName = ".netflow-probe.yaml"

type session struct {
	cancel context.CancelFunc
	src    capture.Source
	table  *flow.Table
	exp    *export.Exporter
}

type gui struct {
	win fyne.Window

	ifaceSel      *widget.Select
	ifaceToDevice map[string]string
	deviceToLabel map[string]string
	collector     *widget.Entry
	port          *widget.Entry
	status        *widget.Label
	logView       *widget.Entry
	startBtn      *widget.Button
	spinner       *kittBar

	sess       *session
	running    bool
	winVisible bool

	exported atomic.Uint64
	packets  atomic.Uint64
	bytes    atomic.Uint64
	logMu    sync.Mutex
	logLines []string
}

func main() {
	selftest := flag.String("selftest", "", "captura unos segundos en la interfaz dada, informa y sale (diagnóstico)")
	flag.Parse()
	if *selftest != "" {
		runSelfTest(*selftest)
		return
	}

	a := app.New()
	a.SetIcon(fyne.NewStaticResource("icon.png", iconPNG))

	g := &gui{}
	g.win = a.NewWindow("Pandora FMS Netflow Probe")
	g.buildUI()
	g.loadConfig()
	g.setupTray(a)
	g.winVisible = true
	g.win.Resize(fyne.NewSize(560, 620))
	g.win.ShowAndRun()
}

// setupTray adds a menu-bar (system tray) icon with quick actions, mirroring
// pandora-mini. Closing the window hides it to the tray instead of quitting.
func (g *gui) setupTray(a fyne.App) {
	desk, ok := a.(desktop.App)
	if !ok {
		return
	}
	desk.SetSystemTrayIcon(fyne.NewStaticResource("tray.png", trayPNG))

	quitItem := fyne.NewMenuItem("Salir", func() { a.Quit() })
	quitItem.IsQuit = true
	desk.SetSystemTrayMenu(fyne.NewMenu("Pandora FMS Netflow Probe",
		fyne.NewMenuItem("Mostrar / ocultar", func() {
			if g.winVisible {
				g.win.Hide()
				g.winVisible = false
			} else {
				g.win.Show()
				g.winVisible = true
			}
		}),
		fyne.NewMenuItem("Iniciar captura", func() {
			if !g.running {
				g.start()
			}
		}),
		fyne.NewMenuItem("Detener captura", func() {
			if g.running {
				g.stop()
			}
		}),
		quitItem,
	))
	g.win.SetCloseIntercept(func() {
		g.win.Hide()
		g.winVisible = false
	})
}

func (g *gui) buildUI() {
	labels, byLabel, byDevice := interfaceOptions()
	g.ifaceToDevice, g.deviceToLabel = byLabel, byDevice
	g.ifaceSel = widget.NewSelect(labels, nil)
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

	g.spinner = newKittBar()
	g.spinner.Hide()

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
		container.NewHBox(widget.NewLabelWithStyle("Estado:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), g.status, g.spinner),
	)

	header := container.NewHBox(
		widget.NewIcon(fyne.NewStaticResource("icon.png", iconPNG)),
		widget.NewLabelWithStyle("Pandora FMS Netflow Probe", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
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

	device := g.ifaceToDevice[g.ifaceSel.Selected]
	if device == "" {
		device = g.ifaceSel.Selected
	}

	cfg := config.Default()
	cfg.Interface = device
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
	g.packets.Store(0)
	g.bytes.Store(0)
	g.spinner.Show()
	g.spinner.Start()
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
				g.packets.Add(1)
				g.bytes.Add(l3)
				table.Observe(key, l3, flags, time.Now())
			}
		}
	}()

	// Exporter goroutine owns cleanup.
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		uiTick := time.NewTicker(time.Second)
		defer uiTick.Stop()
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
					g.spinner.Stop()
					g.spinner.Hide()
					g.setRunningUI(false)
					g.updateStatus()
				})
				return
			case <-uiTick.C:
				g.updateStatus()
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
		active := 0
		if g.sess != nil {
			active = g.sess.table.Len()
		}
		txt := fmt.Sprintf("Capturando · %d paquetes · %s · %d flujos activos · %d exportados",
			g.packets.Load(), humanBytes(g.bytes.Load()), active, g.exported.Load())
		fyne.Do(func() { g.status.SetText(txt) })
		return
	}
	fyne.Do(func() { g.status.SetText(fmt.Sprintf("Detenido · %d flujos exportados", g.exported.Load())) })
}

// humanBytes formats a byte count with a binary unit.
func humanBytes(n uint64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := uint64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGTPE"[exp])
}

// --- permissions (macOS) -----------------------------------------------------

// showPermissionDialog explains the macOS capture permission and offers to
// grant /dev/bpf access with an administrator prompt. There is deliberately no
// "open System Settings" action: macOS has no Privacy toggle for packet
// capture, so sending the user there is a dead end.
func (g *gui) showPermissionDialog(cause error) {
	var msg string
	if cause != nil {
		msg = fmt.Sprintf("No se pudo iniciar la captura:\n\n%v\n\n", cause)
	}
	msg += "Capturar tráfico en macOS necesita acceso a los dispositivos /dev/bpf. " +
		"macOS no tiene una casilla para esto en Ajustes del Sistema: se concede con " +
		"tu contraseña de administrador (una sola vez; se instala un pequeño servicio " +
		"para que se mantenga al reiniciar)."

	label := widget.NewLabel(msg)
	label.Wrapping = fyne.TextWrapWord

	adminBtn := widget.NewButton("Conceder acceso a la captura", func() { g.installCapturePermission() })
	adminBtn.Importance = widget.HighImportance

	content := container.NewVBox(label, layout.NewSpacer(), adminBtn)
	d := dialog.NewCustom("Permisos de captura", "Cerrar", content, g.win)
	d.Resize(fyne.NewSize(500, 300))
	d.Show()
}

const (
	bpfDaemonLabel = "es.artica.netflowprobe.chmodbpf"
	bpfDaemonPath  = "/Library/LaunchDaemons/es.artica.netflowprobe.chmodbpf.plist"
)

// installCapturePermission grants access to the /dev/bpf* capture devices by
// installing a small LaunchDaemon (root) that keeps them accessible to the
// current user, also after a reboot. It is the macOS-correct way; there is no
// Privacy & Security toggle for packet capture.
func (g *gui) installCapturePermission() {
	plist := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key><string>` + bpfDaemonLabel + `</string>
  <key>ProgramArguments</key>
  <array>
    <string>/bin/sh</string>
    <string>-c</string>
    <string>/bin/chmod o+rw /dev/bpf* 2>/dev/null || true</string>
  </array>
  <key>RunAtLoad</key><true/>
  <key>WatchPaths</key><array><string>/dev/bpf0</string></array>
</dict>
</plist>
`
	b64 := base64.StdEncoding.EncodeToString([]byte(plist))
	// No double quotes in this command: it is embedded in an AppleScript string.
	cmd := fmt.Sprintf(
		"/bin/echo %s | /usr/bin/base64 -D > %s && /usr/sbin/chown root:wheel %s && /bin/chmod 644 %s && "+
			"(/bin/launchctl bootout system %s 2>/dev/null; /bin/launchctl bootstrap system %s 2>/dev/null || /bin/launchctl load -w %s 2>/dev/null || true); "+
			"/bin/chmod o+rw /dev/bpf* 2>/dev/null || true",
		b64, bpfDaemonPath, bpfDaemonPath, bpfDaemonPath, bpfDaemonPath, bpfDaemonPath, bpfDaemonPath,
	)
	script := `do shell script "` + cmd + `" with administrator privileges`

	out, err := exec.Command("osascript", "-e", script).CombinedOutput()
	if err != nil {
		g.logf("No se pudo conceder el acceso: %v", err)
		g.showError("No se pudo conceder el acceso.\n\nPuedes ejecutarlo manualmente en Terminal:\n\nsudo chmod o+rw /dev/bpf*")
		return
	}
	if s := strings.TrimSpace(string(out)); s != "" {
		g.logf("%s", s)
	}
	g.logf("Acceso a la captura concedido (servicio %s). Reintentando…", bpfDaemonLabel)
	// Retry automatically so the user does not have to press Iniciar again.
	fyne.Do(func() { g.start() })
}

// runSelfTest performs a short capture and reports whether it works. It is used
// by `-selftest <iface>` for support/diagnostics, without opening the UI.
func runSelfTest(iface string) {
	src, err := capture.OpenLive(iface, "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "FALLO: no se pudo abrir la captura en %s:\n%v\n", iface, err)
		fmt.Fprintf(os.Stderr, "\nSi es un problema de permisos, ejecuta:\n  sudo chmod o+rw /dev/bpf*\n")
		os.Exit(1)
	}
	defer func() { _ = src.Close() }()

	link := src.LinkType()
	deadline := time.Now().Add(3 * time.Second)
	var pkts, byts uint64
	for time.Now().Before(deadline) {
		data, _, err := src.Next()
		if err != nil {
			break
		}
		if _, l3, _, ok := flow.Observe(data, link); ok {
			pkts++
			byts += l3
		}
	}
	fmt.Printf("OK: captura operativa en %s — %d paquetes, %s en 3 s\n", iface, pkts, humanBytes(byts))
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
		if label, ok := g.deviceToLabel[cfg.Interface]; ok {
			g.ifaceSel.SetSelected(label)
		}
	}
	if g.ifaceSel.Selected == "" {
		if label, ok := g.deviceToLabel["en0"]; ok {
			g.ifaceSel.SetSelected(label)
		}
	}
	if g.ifaceSel.Selected == "" && len(g.ifaceSel.Options) > 0 {
		g.ifaceSel.SetSelected(g.ifaceSel.Options[0])
	}
	g.updateStatus()
}

func (g *gui) saveConfig() error {
	port, err := strconv.Atoi(strings.TrimSpace(g.port.Text))
	if err != nil {
		return fmt.Errorf("puerto inválido")
	}
	device := g.ifaceToDevice[g.ifaceSel.Selected]
	if device == "" {
		device = g.ifaceSel.Selected
	}
	cfg := config.Default()
	cfg.Interface = device
	cfg.Collector = strings.TrimSpace(g.collector.Text)
	cfg.Port = port
	return cfg.Save(g.configPath())
}

// interfaceOptions returns the display labels of the interfaces that are
// actually usable, plus the label<->device maps. Only interfaces with a
// non-link-local address are shown, so macOS internals (anpi, awdl, llw,
// bridge, idle utun...) do not clutter the list. Each label carries a friendly
// description (Wi-Fi, Thunderbolt 1, VPN, ...) and the address.
func interfaceOptions() (labels []string, byLabel, byDevice map[string]string) {
	byLabel = map[string]string{}
	byDevice = map[string]string{}

	ports := hardwarePorts()
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, byLabel, byDevice
	}

	type item struct {
		label    string
		device   string
		physical bool
	}
	var items []item
	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		ip, ok := usableAddr(iface)
		if !ok {
			continue
		}
		desc := ports[iface.Name]
		if desc == "" {
			desc = kindOfInterface(iface.Name)
		}
		items = append(items, item{
			label:    interfaceLabel(desc, iface.Name, ip),
			device:   iface.Name,
			physical: ports[iface.Name] != "",
		})
	}

	// Physical ports first (Wi-Fi, Ethernet...), then VPN/bridges; by name.
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].physical != items[j].physical {
			return items[i].physical
		}
		return items[i].device < items[j].device
	})

	for _, it := range items {
		labels = append(labels, it.label)
		byLabel[it.label] = it.device
		byDevice[it.device] = it.label
	}
	return labels, byLabel, byDevice
}

func interfaceLabel(desc, name string, ip net.IP) string {
	switch {
	case desc != "" && strings.Contains(desc, name):
		return fmt.Sprintf("%s · %s", desc, ip)
	case desc != "":
		return fmt.Sprintf("%s (%s) · %s", desc, name, ip)
	default:
		return fmt.Sprintf("%s · %s", name, ip)
	}
}

// kindOfInterface gives a generic description when macOS reports no hardware
// port for the device.
func kindOfInterface(name string) string {
	switch {
	case strings.HasPrefix(name, "utun"), strings.HasPrefix(name, "ipsec"),
		strings.HasPrefix(name, "tun"), strings.HasPrefix(name, "tap"),
		strings.HasPrefix(name, "ppp"):
		return "VPN"
	case strings.HasPrefix(name, "bridge"):
		return "Bridge"
	case strings.HasPrefix(name, "en"):
		return "Ethernet"
	default:
		return ""
	}
}

// usableAddr returns an interface address that indicates real connectivity,
// preferring IPv4. Loopback and link-local (169.254/16, fe80::/10) are ignored,
// as are IPv6 ULA addresses when there is no IPv4.
func usableAddr(iface net.Interface) (net.IP, bool) {
	addrs, err := iface.Addrs()
	if err != nil {
		return nil, false
	}
	var v6 net.IP
	for _, a := range addrs {
		n, ok := a.(*net.IPNet)
		if !ok {
			continue
		}
		ip := n.IP
		if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
			continue
		}
		if ip.To4() != nil {
			return ip, true
		}
		if ip.IsGlobalUnicast() && !ip.IsPrivate() && v6 == nil {
			v6 = ip
		}
	}
	if v6 != nil {
		return v6, true
	}
	return nil, false
}

// hardwarePorts maps a BSD device name (en0, en1, ...) to the friendly hardware
// port name reported by macOS ("Wi-Fi", "Thunderbolt 1", ...). Best effort.
func hardwarePorts() map[string]string {
	out, err := exec.Command("networksetup", "-listallhardwareports").Output()
	if err != nil {
		return nil
	}
	ports := map[string]string{}
	var current string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "Hardware Port:"):
			current = strings.TrimSpace(strings.TrimPrefix(line, "Hardware Port:"))
		case strings.HasPrefix(line, "Device:"):
			dev := strings.TrimSpace(strings.TrimPrefix(line, "Device:"))
			if dev != "" && current != "" {
				ports[dev] = current
			}
		}
	}
	return ports
}
