//go:build !(darwin || windows || linux) || !cgo

// The graphical front-end is available on macOS, Windows and Linux and requires
// cgo (Fyne). On other platforms or without cgo this stub keeps the package
// buildable.
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "netflow-probe-gui: la interfaz grafica solo esta disponible en macOS.")
	fmt.Fprintln(os.Stderr, "Usa el CLI 'netflow-probe' en esta plataforma.")
	os.Exit(1)
}
