//go:build windows && cgo

package main

import (
	"fmt"
	"os"

	"fyne.io/fyne/v2"
)

// WinDivert captures the whole host, so there is no interface to choose.
const platformSelectsInterface = false

func platformPermissionHint() string {
	return "En Windows la captura la realiza el driver WinDivert, que necesita " +
		"privilegios de administrador. Reinicia la aplicación como administrador " +
		"(Windows pedirá confirmación) o vuelve a abrirla con «Ejecutar como " +
		"administrador»."
}

func platformPermissionButton() string { return "Reiniciar como administrador" }

// requestPlatformPermission relaunches the app elevated via PowerShell's
// Start-Process -Verb RunAs (UAC prompt), then quits this instance.
func (g *gui) requestPlatformPermission() {
	exe, err := os.Executable()
	if err != nil {
		g.showError("No se pudo localizar el ejecutable.")
		return
	}
	cmd := newCommand("powershell", "-NoProfile", "-Command",
		fmt.Sprintf("Start-Process -Verb RunAs -FilePath '%s'", exe))
	if err := cmd.Start(); err != nil {
		g.logf("No se pudo reiniciar como administrador: %v", err)
		g.showError("No se pudo reiniciar como administrador.\n\nCierra la aplicación y ábrela de nuevo con «Ejecutar como administrador».")
		return
	}
	g.logf("Reiniciando como administrador…")
	if g.app != nil {
		fyne.Do(func() { g.app.Quit() })
	}
}
