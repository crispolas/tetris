//go:build !windows

package main

import (
	"os"
	"syscall"
	"unsafe"
)

// ===== SUPORTE CROSS-PLATFORM: UNIX (Linux / macOS) =====
// Este arquivo e compilado em qualquer sistema que nao seja Windows.
// Usa syscall POSIX (termios) para configurar o terminal em modo raw:
//   - Desativa modo canonico (ICANON): leitura imediata por tecla, sem buffer de linha
//   - Desativa eco (ECHO): teclas digitadas nao aparecem na tela
//   - Desativa sinais de controle (ISIG): Ctrl+C nao encerra o processo automaticamente
//   - Desativa controle de fluxo (IXON): Ctrl+S/Q nao congela o terminal
//   - Desativa pos-processamento (OPOST): '\n' nao e convertido em '\r\n'

// Estrutura termios compativel com Linux e macOS (64-bit)
type termios struct {
	Iflag  uint32
	Oflag  uint32
	Cflag  uint32
	Lflag  uint32
	Cc     [20]uint8
	Ispeed uint32
	Ospeed uint32
}

// Constantes POSIX para manipulacao de flags de terminal
const (
	TCGETS = 0x5401 // ioctl: obter configuracao atual do terminal
	TCSETS = 0x5402 // ioctl: aplicar nova configuracao ao terminal

	ICANON = 0x0002 // modo canonico (bufferizacao por linha)
	ECHO   = 0x0008 // eco de caracteres na tela
	ISIG   = 0x0001 // geracao de sinais por Ctrl+C, Ctrl+Z
	IXON   = 0x0400 // controle de fluxo por software (Ctrl+S / Ctrl+Q)
	OPOST  = 0x0001 // pos-processamento de saida (\n -> \r\n)
)

var directInputEnabled bool
var originalTermios termios

func enableDirectInput() {
	var current termios

	// Leitura da configuracao atual do terminal via syscall ioctl
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(os.Stdin.Fd()),
		TCGETS,
		uintptr(unsafe.Pointer(&current)),
	)
	if errno != 0 {
		return
	}

	originalTermios = current

	// Aritmetica de bits: desativa flags indesejados sem alterar os demais
	current.Lflag = current.Lflag &^ uint32(ICANON|ECHO|ISIG)
	current.Iflag = current.Iflag &^ uint32(IXON)
	current.Oflag = current.Oflag &^ uint32(OPOST)

	// VMIN=1: retorna ao ler pelo menos 1 byte; VTIME=0: sem timeout (nao bloqueante)
	current.Cc[6] = 1 // VMIN
	current.Cc[5] = 0 // VTIME

	// Aplica nova configuracao no terminal
	_, _, errno = syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(os.Stdin.Fd()),
		TCSETS,
		uintptr(unsafe.Pointer(&current)),
	)
	if errno != 0 {
		return
	}

	directInputEnabled = true
}

func restoreTerminalInput() {
	if !directInputEnabled {
		return
	}

	// Restaura as configuracoes originais capturadas antes da modificacao
	syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(os.Stdin.Fd()),
		TCSETS,
		uintptr(unsafe.Pointer(&originalTermios)),
	)
	directInputEnabled = false
}
