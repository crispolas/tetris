package main

import (
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"
)

const boardHeight = 18
const boardWidth = 14
const pieceSize = 4
const stdInputHandle = -10
const enableEchoInput = 0x0004
const enableLineInput = 0x0002
const emptyBlock = "  "

// Matriz global do jogo: 0 significa vazio; 1 a 7 indicam blocos fixos coloridos.
// O tabuleiro e alterado diretamente pelas funcoes.
// Esta escolha reforca o paradigma imperativo: estado global simples + mutacao explicita.
var board [boardHeight][boardWidth]int

// Estado global simples da partida.
var currentPiece [pieceSize][pieceSize]int
var currentColor int
var currentX int
var currentY int
var score int
var gameOver bool
var quitGame bool
var seed int
var directInputEnabled bool
var stdinHandle syscall.Handle
var originalConsoleMode uint32

var kernel32 = syscall.NewLazyDLL("kernel32.dll")
var procSetConsoleMode = kernel32.NewProc("SetConsoleMode")

// Pecas basicas do Tetris representadas como matrizes simples 4x4.
// Nao ha objetos, metodos, interfaces, entidades ou componentes.
var pieces = [7][pieceSize][pieceSize]int{
	{
		{0, 0, 0, 0},
		{1, 1, 1, 1},
		{0, 0, 0, 0},
		{0, 0, 0, 0},
	},
	{
		{1, 1, 0, 0},
		{1, 1, 0, 0},
		{0, 0, 0, 0},
		{0, 0, 0, 0},
	},
	{
		{0, 1, 0, 0},
		{1, 1, 1, 0},
		{0, 0, 0, 0},
		{0, 0, 0, 0},
	},
	{
		{0, 1, 1, 0},
		{1, 1, 0, 0},
		{0, 0, 0, 0},
		{0, 0, 0, 0},
	},
	{
		{1, 1, 0, 0},
		{0, 1, 1, 0},
		{0, 0, 0, 0},
		{0, 0, 0, 0},
	},
	{
		{1, 0, 0, 0},
		{1, 1, 1, 0},
		{0, 0, 0, 0},
		{0, 0, 0, 0},
	},
	{
		{0, 0, 1, 0},
		{1, 1, 1, 0},
		{0, 0, 0, 0},
		{0, 0, 0, 0},
	},
}

func main() {
	commandChannel := make(chan rune, 20)

	initBoard()
	seed = int(time.Now().UnixNano() % 100000)
	spawnPiece()
	enableDirectInput()
	defer restoreTerminalInput()

	go readCommands(commandChannel)

	for !gameOver && !quitGame {
		drawBoard()
		processPendingCommands(commandChannel)

		// ===== CONCEITO DA DISCIPLINA: PASSAGEM POR REFERENCIA =====
		// Aqui a matriz global e enviada por ponteiro para ser alterada diretamente.
		// A funcao gameTick recebe o endereco de board e muda a matriz original.
		gameTick(&board)

		time.Sleep(500 * time.Millisecond)
	}

	restoreTerminalInput()
	drawBoard()
	if gameOver {
		fmt.Println("FIM DE JOGO! Pressione Enter para encerrar.")
		fmt.Scanln()
	}
}

func initBoard() {
	for row := 0; row < boardHeight; row++ {
		for col := 0; col < boardWidth; col++ {
			board[row][col] = 0
		}
	}

	score = 0
	gameOver = false
	quitGame = false
	directInputEnabled = false
}

func drawBoard() {
	var display [boardHeight][boardWidth]int
	screen := ""
	border := "+" + strings.Repeat("--", boardWidth) + "+\n"
	footer := "+" + strings.Repeat("--", boardWidth) + "+\n"

	for row := 0; row < boardHeight; row++ {
		for col := 0; col < boardWidth; col++ {
			display[row][col] = board[row][col]
		}
	}

	for row := 0; row < pieceSize; row++ {
		for col := 0; col < pieceSize; col++ {
			if currentPiece[row][col] == 1 {
				boardRow := currentY + row
				boardCol := currentX + col

				if boardRow >= 0 && boardRow < boardHeight && boardCol >= 0 && boardCol < boardWidth {
					display[boardRow][boardCol] = currentColor
				}
			}
		}
	}

	clearTerminal()
	screen = screen + "TETRIS IMPERATIVO EM GO\n"
	screen = screen + "Controles: A esquerda | D direita | S descer | W girar | Q sair\n"
	if directInputEnabled {
		screen = screen + "Entrada direta ativada: aperte a tecla, sem Enter. Pontuacao: " + fmt.Sprint(score) + "\n"
	} else {
		screen = screen + "Entrada direta indisponivel: use a tecla e Enter. Pontuacao: " + fmt.Sprint(score) + "\n"
	}
	screen = screen + border

	for row := 0; row < boardHeight; row++ {
		screen = screen + "|"
		for col := 0; col < boardWidth; col++ {
			cell := display[row][col]
			screen = screen + colorBlock(cell)
		}
		screen = screen + "|\n"
	}

	screen = screen + footer
	fmt.Print(screen)
}

func clearTerminal() {
	fmt.Print("\033[H\033[2J")
}

func colorBlock(color int) string {
	switch color {
	case 1:
		return "\033[46m  \033[0m" // ciano
	case 2:
		return "\033[43m  \033[0m" // amarelo
	case 3:
		return "\033[45m  \033[0m" // magenta
	case 4:
		return "\033[42m  \033[0m" // verde
	case 5:
		return "\033[41m  \033[0m" // vermelho
	case 6:
		return "\033[44m  \033[0m" // azul
	case 7:
		return "\033[47m  \033[0m" // branco
	default:
		return emptyBlock
	}
}

func readCommands(commandChannel chan rune) {
	buffer := make([]byte, 1)

	for {
		bytesRead, err := os.Stdin.Read(buffer)
		if err != nil {
			return
		}

		if bytesRead > 0 {
			sendCommand(commandChannel, rune(buffer[0]))
		}
	}
}

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

func sendCommand(commandChannel chan rune, command rune) {
	select {
	case commandChannel <- command:
	default:
		// Se o jogador digitar muitos comandos rapidamente, descartamos o excesso.
		// Isso evita acumulo de entrada e melhora a estabilidade no terminal.
	}
}

func processPendingCommands(commandChannel chan rune) {
	for {
		select {
		case command := <-commandChannel:
			processCommand(command)
		default:
			return
		}
	}
}

func processCommand(command rune) {
	switch command {
	case 'a', 'A':
		movePiece(-1, 0)
	case 'd', 'D':
		movePiece(1, 0)
	case 's', 'S':
		movePiece(0, 1)
	case 'w', 'W':
		rotatePiece()
	case 'q', 'Q':
		quitGame = true
	}
}

func spawnPiece() {
	index := nextPieceIndex()
	currentColor = index + 1

	for row := 0; row < pieceSize; row++ {
		for col := 0; col < pieceSize; col++ {
			currentPiece[row][col] = pieces[index][row][col]
		}
	}

	currentX = boardWidth/2 - 2
	currentY = 0

	if !canPlace(currentPiece, currentX, currentY, &board) {
		gameOver = true
	}
}

func nextPieceIndex() int {
	// Gerador simples e imperativo para evitar dependencias ou abstracoes extras.
	seed = (seed*1103515245 + 12345) & 0x7fffffff
	return seed % 7
}

func canMove(dx int, dy int) bool {
	return canPlace(currentPiece, currentX+dx, currentY+dy, &board)
}

func canPlace(piece [pieceSize][pieceSize]int, pieceX int, pieceY int, gameBoard *[boardHeight][boardWidth]int) bool {
	// Colisao linha por linha: cada posicao ocupada da peca e testada contra
	// limites da matriz e blocos ja fixados no tabuleiro.
	for row := 0; row < pieceSize; row++ {
		for col := 0; col < pieceSize; col++ {
			if piece[row][col] == 1 {
				boardRow := pieceY + row
				boardCol := pieceX + col

				if boardRow < 0 || boardRow >= boardHeight {
					return false
				}

				if boardCol < 0 || boardCol >= boardWidth {
					return false
				}

				if (*gameBoard)[boardRow][boardCol] != 0 {
					return false
				}
			}
		}
	}

	return true
}

func movePiece(dx int, dy int) {
	if canMove(dx, dy) {
		currentX = currentX + dx
		currentY = currentY + dy
	}
}

func rotatePiece() {
	var rotated [pieceSize][pieceSize]int

	for row := 0; row < pieceSize; row++ {
		for col := 0; col < pieceSize; col++ {
			rotated[col][pieceSize-1-row] = currentPiece[row][col]
		}
	}

	if canPlace(rotated, currentX, currentY, &board) {
		for row := 0; row < pieceSize; row++ {
			for col := 0; col < pieceSize; col++ {
				currentPiece[row][col] = rotated[row][col]
			}
		}
	}
}

func applyGravity(gameBoard *[boardHeight][boardWidth]int) {
	// Gravidade procedural aplicada a cada tick:
	// primeiro tenta descer uma linha; se nao puder, fixa a peca na matriz.
	if canMove(0, 1) {
		currentY = currentY + 1
	} else {
		lockPiece(gameBoard)
		clearLines(gameBoard)
		spawnPiece()
	}
}

func lockPiece(gameBoard *[boardHeight][boardWidth]int) {
	// MUTACAO EXPLICITA DA MATRIZ:
	// cada bloco da peca atual vira bloco fixo no tabuleiro original.
	for row := 0; row < pieceSize; row++ {
		for col := 0; col < pieceSize; col++ {
			if currentPiece[row][col] == 1 {
				boardRow := currentY + row
				boardCol := currentX + col

				if boardRow >= 0 && boardRow < boardHeight && boardCol >= 0 && boardCol < boardWidth {
					(*gameBoard)[boardRow][boardCol] = currentColor
				}
			}
		}
	}
}

func clearLines(gameBoard *[boardHeight][boardWidth]int) {
	linesCleared := 0

	for row := boardHeight - 1; row >= 0; row-- {
		fullLine := true

		for col := 0; col < boardWidth; col++ {
			if (*gameBoard)[row][col] == 0 {
				fullLine = false
				break
			}
		}

		if fullLine {
			linesCleared = linesCleared + 1

			for moveRow := row; moveRow > 0; moveRow-- {
				for col := 0; col < boardWidth; col++ {
					(*gameBoard)[moveRow][col] = (*gameBoard)[moveRow-1][col]
				}
			}

			for col := 0; col < boardWidth; col++ {
				(*gameBoard)[0][col] = 0
			}

			row = row + 1
		}
	}

	if linesCleared > 0 {
		score = score + linesCleared*100
	}
}

func gameTick(gameBoard *[boardHeight][boardWidth]int) {
	// ===== CONCEITO DA DISCIPLINA: PASSAGEM POR REFERENCIA =====
	// gameBoard e um ponteiro para a matriz global board.
	// Portanto, lockPiece e clearLines modificam diretamente a matriz original,
	// sem criar copia do tabuleiro.
	if gameOver || quitGame {
		return
	}

	// Controle sequencial imperativo: processa um passo da gravidade,
	// depois as funcoes chamadas podem fixar peca, limpar linhas e criar nova peca.
	applyGravity(gameBoard)
}
