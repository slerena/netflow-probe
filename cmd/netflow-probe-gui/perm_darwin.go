//go:build darwin && cgo

package main

import (
	"encoding/base64"
	"fmt"
	"os/exec"
	"strings"

	"fyne.io/fyne/v2"
)

// macOS captures through /dev/bpf*, so the interface must be chosen.
const platformSelectsInterface = true

func platformPermissionHint() string {
	return "Capturar tráfico en macOS necesita acceso a los dispositivos /dev/bpf. " +
		"macOS no tiene una casilla para esto en Ajustes del Sistema: se concede con " +
		"tu contraseña de administrador (una sola vez; se instala un pequeño servicio " +
		"para que se mantenga al reiniciar)."
}

func platformPermissionButton() string { return "Conceder acceso a la captura" }

const (
	bpfDaemonLabel = "es.artica.netflowprobe.chmodbpf"
	bpfDaemonPath  = "/Library/LaunchDaemons/es.artica.netflowprobe.chmodbpf.plist"
)

// requestPlatformPermission grants access to the /dev/bpf* capture devices by
// installing a small LaunchDaemon (root) that keeps them accessible to the
// current user, also after a reboot, then retries the capture.
func (g *gui) requestPlatformPermission() {
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
	fyne.Do(func() { g.start() })
}
