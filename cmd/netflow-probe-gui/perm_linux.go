//go:build linux && cgo

package main

import (
	"os"
	"os/exec"
	"strings"

	"fyne.io/fyne/v2"
)

// Linux captures through AF_PACKET, so the interface must be chosen.
const platformSelectsInterface = true

func platformPermissionHint() string {
	return "Capturar tráfico en Linux necesita el permiso CAP_NET_RAW sobre el " +
		"ejecutable (sockets AF_PACKET). Se concede una sola vez con tu contraseña " +
		"(pkexec/sudo) ejecutando:\n\n" +
		"    setcap cap_net_raw+ep <binario>\n\n" +
		"También puedes ejecutar la aplicación como root."
}

func platformPermissionButton() string { return "Conceder acceso a la captura (setcap)" }

// requestPlatformPermission runs setcap cap_net_raw+ep on the executable via
// pkexec (graphical prompt) or sudo (terminal prompt), then retries capture.
func (g *gui) requestPlatformPermission() {
	exe, err := os.Executable()
	if err != nil {
		g.showError("No se pudo localizar el ejecutable.")
		return
	}

	var cmd *exec.Cmd
	if _, err := exec.LookPath("pkexec"); err == nil {
		cmd = exec.Command("pkexec", "setcap", "cap_net_raw+ep", exe)
	} else {
		cmd = exec.Command("sudo", "setcap", "cap_net_raw+ep", exe)
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		g.logf("No se pudo conceder el permiso: %v", err)
		g.showError("No se pudo conceder el permiso.\n\nEjecútalo manualmente en una terminal:\n\nsudo setcap cap_net_raw+ep \"" + exe + "\"\n\n(Necesita el paquete libcap: libcap2-bin en Debian/Ubuntu, libcap en Fedora/Rocky.)")
		return
	}
	if s := strings.TrimSpace(string(out)); s != "" {
		g.logf("%s", s)
	}
	g.logf("Permiso CAP_NET_RAW concedido. Reintentando…")
	fyne.Do(func() { g.start() })
}
