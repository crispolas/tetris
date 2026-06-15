//go:build windows

package main

import "syscall"

// ===== SUPORTE CROSS-PLATFORM: WINDOWS =====
// Este arquivo e compilado apenas em Windows (build tag: //go:build windows).
// Usa a Win32 API via syscall para desativar o modo canonico e o eco de entrada.

const stdInputHandle = -10
const enableEchoInput = 0x0004
const enableLineInput = 0x0002

var directInputEnabled bool
var stdinHandle syscall.Handle
var originalConsoleMode uint32

var kernel32 = syscall.NewLazyDLL("kernel32.dll")
var procSetConsoleMode = kernel32.NewProc("SetConsoleMode")

func enableDirectInput() {
	var mode uint32
	handle, err := syscall.GetStdHandle(stdInputHandle)
	if err != nil {
		return
	}

	err = syscall.GetConsoleMode(handle, &mode)
	if err != nil {
		return
	}

	stdinHandle = handle
	originalConsoleMode = mode

	// Desativa bufferizacao por linha e eco de caracteres via bitmask
	mode = mode &^ enableLineInput
	mode = mode &^ enableEchoInput

	result, _, _ := procSetConsoleMode.Call(uintptr(stdinHandle), uintptr(mode))
	if result != 0 {
		directInputEnabled = true
	}
}

func restoreTerminalInput() {
	if directInputEnabled {
		procSetConsoleMode.Call(uintptr(stdinHandle), uintptr(originalConsoleMode))
		directInputEnabled = false
	}
}
