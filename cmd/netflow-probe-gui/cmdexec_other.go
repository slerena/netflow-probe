//go:build !windows

package main

import "syscall"

// En macOS y Linux no hay ventanas de consola que ocultar.
func hiddenSysProcAttr() *syscall.SysProcAttr { return nil }
