package main

import (
	"context"
	"os/exec"
)

// newCommand lanza un proceso hijo SIN mostrar su ventana.
//
// En Windows, una aplicación de GUI (subsistema windowsgui) que arranca un
// programa de consola —powershell, cmd, redes…— hace que Windows le cree una
// consola nueva, y eso se ve como una ventana que aparece y desaparece.
// hiddenSysProcAttr() pide CREATE_NO_WINDOW (+ HideWindow). En macOS y Linux es
// exec.Command normal.
func newCommand(name string, args ...string) *exec.Cmd {
	c := exec.Command(name, args...)
	c.SysProcAttr = hiddenSysProcAttr()
	return c
}

// newCommandContext es como newCommand pero con contexto (cancelación).
func newCommandContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	c := exec.CommandContext(ctx, name, args...)
	c.SysProcAttr = hiddenSysProcAttr()
	return c
}
