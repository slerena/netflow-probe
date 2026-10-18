//go:build windows && cgo

package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"unsafe"

	"github.com/go-gl/glfw/v3.4/glfw"
)

// compatEnvVar marca el proceso hijo lanzado en modo compatible, para no
// relanzar en bucle si tampoco allí hubiera OpenGL.
const compatEnvVar = "NETFLOW_PROBE_COMPAT"

var (
	user32  = syscall.NewLazyDLL("user32.dll")
	msgBoxW = user32.NewProc("MessageBoxW")
)

// glUsable indica si el equipo puede crear un contexto OpenGL 2.1, el mínimo
// que exige Fyne. En Windows el fallo es MUDO (la app arranca, no crea ventana y
// termina), así que se comprueba antes de app.New().
//
// Crea un contexto propio (ventana oculta) y lo destruye, dejando GLFW como
// estaba para que el Init posterior de Fyne parta de cero. Fyne ya llama a
// runtime.LockOSThread en su init(); el Lock/Unlock de aquí mantiene fijo el
// hilo entre Init y Terminate (GLFW exige el mismo hilo).
func glUsable() bool {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if err := glfw.Init(); err != nil {
		return false
	}
	defer func() {
		glfw.DefaultWindowHints()
		glfw.Terminate()
	}()

	glfw.WindowHint(glfw.Visible, glfw.False)
	glfw.WindowHint(glfw.ContextVersionMajor, 2)
	glfw.WindowHint(glfw.ContextVersionMinor, 1)

	win, err := glfw.CreateWindow(64, 64, "gl-check", nil, nil)
	if err != nil || win == nil {
		return false
	}
	win.MakeContextCurrent()
	major := win.GetAttrib(glfw.ContextVersionMajor)
	minor := win.GetAttrib(glfw.ContextVersionMinor)
	win.Destroy()

	return major > 2 || (major == 2 && minor >= 1)
}

// tryCompatRelaunch arranca la copia del programa que va junto a Mesa (swgl\)
// y devuelve true si lo consiguió. Es el relevo automático cuando el equipo no
// ofrece OpenGL 2.1: el usuario hace doble clic en el ejecutable normal y la
// aplicación funciona, sin lanzadores ni pasos manuales.
func tryCompatRelaunch() bool {
	if os.Getenv(compatEnvVar) != "" {
		return false // ya somos el proceso en modo compatible
	}

	exe, err := os.Executable()
	if err != nil {
		return false
	}
	dir := filepath.Dir(exe)
	compatDir := filepath.Join(dir, "swgl")
	child := filepath.Join(compatDir, filepath.Base(exe))

	if _, err := os.Stat(child); err != nil {
		return false
	}
	if _, err := os.Stat(filepath.Join(compatDir, "opengl32.dll")); err != nil {
		return false
	}

	env := make([]string, 0, len(os.Environ())+4)
	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, "GALLIUM_DRIVER=") ||
			strings.HasPrefix(kv, "MESA_GL_VERSION_OVERRIDE=") ||
			strings.HasPrefix(kv, "MESA_GLSL_VERSION_OVERRIDE=") ||
			strings.HasPrefix(kv, compatEnvVar+"=") {
			continue
		}
		env = append(env, kv)
	}
	env = append(env,
		"GALLIUM_DRIVER=llvmpipe",
		"MESA_GL_VERSION_OVERRIDE=3.3",
		"MESA_GLSL_VERSION_OVERRIDE=330",
		compatEnvVar+"=1",
	)

	cmd := newCommand(child)
	cmd.Dir = compatDir
	cmd.Env = env
	if err := cmd.Start(); err != nil {
		return false
	}
	return true
}

// showGLWarning muestra el aviso con MessageBoxW (no necesita OpenGL) cuando no
// hay ni OpenGL 2.1 ni modo compatible disponible.
func showGLWarning() {
	const (
		mbOK            = 0x00000000
		mbIconWarning   = 0x00000030
		mbSetForeground = 0x00010000
		mbTopmost       = 0x00040000
	)
	text, err1 := syscall.UTF16PtrFromString(
		"Pandora FMS Netflow Probe no puede dibujar en este equipo:\n\n" +
			"no hay OpenGL 2.1 y no se encontró el modo compatible (carpeta swgl\\ " +
			"junto al ejecutable).\n\n" +
			"Instala el paquete completo o usa un equipo con OpenGL 2.1.")
	title, err2 := syscall.UTF16PtrFromString("Pandora FMS Netflow Probe")
	if err1 != nil || err2 != nil {
		return
	}
	_, _, _ = msgBoxW.Call(0,
		uintptr(unsafe.Pointer(text)),
		uintptr(unsafe.Pointer(title)),
		uintptr(mbOK|mbIconWarning|mbSetForeground|mbTopmost))
}
