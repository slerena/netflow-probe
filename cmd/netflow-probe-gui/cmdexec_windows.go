//go:build windows

package main

import "syscall"

// createNoWindow es el flag CREATE_NO_WINDOW de CreateProcess: el hijo no recibe
// consola, así que no aparece ninguna ventana. HideWindow es el cinturón
// (STARTF_USESHOWWINDOW + SW_HIDE) por si el programa respeta una cosa y no la otra.
const createNoWindow = 0x08000000

func hiddenSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: createNoWindow,
	}
}
