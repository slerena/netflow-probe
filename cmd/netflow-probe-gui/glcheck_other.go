//go:build !windows || !cgo

package main

// En macOS y Linux Fyne enlaza el OpenGL del sistema y no hay comprobación
// previa que hacer: si faltara, el arranque no es silencioso.
func glUsable() bool { return true }

// showGLWarning solo tiene sentido en Windows (MessageBox nativo).
func showGLWarning() {}

// tryCompatRelaunch: el modo compatible con Mesa solo existe en Windows.
func tryCompatRelaunch() bool { return false }
