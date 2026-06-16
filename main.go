package main

import (
	"fmt"
	"image/color"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const boardHeight = 18
const boardWidth = 14
const pieceSize = 4
const cellSize = 34
const padding = 16
const sidePanel = 180

const screenW = boardWidth*cellSize + padding*2 + sidePanel
const screenH = boardHeight*cellSize + padding*2

// ===== ESTADO GLOBAL (paradigma imperativo) =====
// Matriz global do jogo: 0 = vazio; 1–7 = blocos fixos coloridos.
// Mutacao explicita via ponteiros — nucleo identico ao original terminal.
var board [boardHeight][boardWidth]int

var currentPiece [pieceSize][pieceSize]int
var currentColor int
var currentX int
var currentY int
var score int
var level int
var linesCleared int
var gameOver bool
var seed int
var tickAccum float64
var tickInterval float64

// Pecas identicas ao original — matrizes 4x4 sem objetos ou interfaces.
var pieces = [7][pieceSize][pieceSize]int{
	{{0, 0, 0, 0}, {1, 1, 1, 1}, {0, 0, 0, 0}, {0, 0, 0, 0}},
	{{1, 1, 0, 0}, {1, 1, 0, 0}, {0, 0, 0, 0}, {0, 0, 0, 0}},
	{{0, 1, 0, 0}, {1, 1, 1, 0}, {0, 0, 0, 0}, {0, 0, 0, 0}},
	{{0, 1, 1, 0}, {1, 1, 0, 0}, {0, 0, 0, 0}, {0, 0, 0, 0}},
	{{1, 1, 0, 0}, {0, 1, 1, 0}, {0, 0, 0, 0}, {0, 0, 0, 0}},
	{{1, 0, 0, 0}, {1, 1, 1, 0}, {0, 0, 0, 0}, {0, 0, 0, 0}},
	{{0, 0, 1, 0}, {1, 1, 1, 0}, {0, 0, 0, 0}, {0, 0, 0, 0}},
}

// Proxima peca — exibida no painel lateral
var nextPiece [pieceSize][pieceSize]int
var nextColor int

// Paleta de cores das pecas
var pieceColors = [8]color.RGBA{
	{0, 0, 0, 0},           // 0 = vazio
	{0, 188, 212, 255},     // 1 ciano   - I
	{255, 193, 7, 255},     // 2 amarelo - O
	{156, 39, 176, 255},    // 3 roxo    - T
	{76, 175, 80, 255},     // 4 verde   - S
	{244, 67, 54, 255},     // 5 vermelho- Z
	{33, 150, 243, 255},    // 6 azul    - J
	{158, 158, 158, 255},   // 7 cinza   - L
}

// Game e a struct minima exigida pelo Ebitengine — sem logica de jogo aqui.
type Game struct{}

func main() {
	ebiten.SetWindowSize(screenW, screenH)
	ebiten.SetWindowTitle("Tetris Imperativo — Go")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeDisabled)

	initGame()

	if err := ebiten.RunGame(&Game{}); err != nil {
		fmt.Println(err)
	}
}

func initGame() {
	for row := 0; row < boardHeight; row++ {
		for col := 0; col < boardWidth; col++ {
			board[row][col] = 0
		}
	}

	score = 0
	level = 1
	linesCleared = 0
	gameOver = false
	tickAccum = 0
	tickInterval = 0.5
	seed = int(time.Now().UnixNano() % 100000)

	// Sorteia proxima peca antes de spawnar a primeira
	ni := nextPieceIndex()
	for r := 0; r < pieceSize; r++ {
		for c := 0; c < pieceSize; c++ {
			nextPiece[r][c] = pieces[ni][r][c]
		}
	}
	nextColor = ni + 1

	spawnPiece()
}

// ===== CONCEITO DA DISCIPLINA: GERADOR LINEAR CONGRUENCIAL =====
func nextPieceIndex() int {
	seed = (seed*1103515245 + 12345) & 0x7fffffff
	return seed % 7
}

func spawnPiece() {
	for r := 0; r < pieceSize; r++ {
		for c := 0; c < pieceSize; c++ {
			currentPiece[r][c] = nextPiece[r][c]
		}
	}
	currentColor = nextColor

	ni := nextPieceIndex()
	for r := 0; r < pieceSize; r++ {
		for c := 0; c < pieceSize; c++ {
			nextPiece[r][c] = pieces[ni][r][c]
		}
	}
	nextColor = ni + 1

	currentX = boardWidth/2 - 2
	currentY = 0

	if !canPlace(currentPiece, currentX, currentY, &board) {
		gameOver = true
	}
}

// ===== CONCEITO DA DISCIPLINA: PASSAGEM POR REFERENCIA =====
func canPlace(piece [pieceSize][pieceSize]int, px int, py int, gameBoard *[boardHeight][boardWidth]int) bool {
	for row := 0; row < pieceSize; row++ {
		for col := 0; col < pieceSize; col++ {
			if piece[row][col] == 1 {
				br := py + row
				bc := px + col
				if br < 0 || br >= boardHeight || bc < 0 || bc >= boardWidth {
					return false
				}
				if (*gameBoard)[br][bc] != 0 {
					return false
				}
			}
		}
	}
	return true
}

func movePiece(dx int, dy int) {
	if canPlace(currentPiece, currentX+dx, currentY+dy, &board) {
		currentX += dx
		currentY += dy
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

func hardDrop() {
	for canPlace(currentPiece, currentX, currentY+1, &board) {
		currentY++
	}
	lockPiece(&board)
	clearFullLines(&board)
	spawnPiece()
}

// ===== CONCEITO DA DISCIPLINA: MUTACAO EXPLICITA VIA PONTEIRO =====
func lockPiece(gameBoard *[boardHeight][boardWidth]int) {
	for row := 0; row < pieceSize; row++ {
		for col := 0; col < pieceSize; col++ {
			if currentPiece[row][col] == 1 {
				br := currentY + row
				bc := currentX + col
				if br >= 0 && br < boardHeight && bc >= 0 && bc < boardWidth {
					(*gameBoard)[br][bc] = currentColor
				}
			}
		}
	}
}

func clearFullLines(gameBoard *[boardHeight][boardWidth]int) {
	cleared := 0
	for row := boardHeight - 1; row >= 0; row-- {
		full := true
		for col := 0; col < boardWidth; col++ {
			if (*gameBoard)[row][col] == 0 {
				full = false
				break
			}
		}
		if full {
			cleared++
			for mr := row; mr > 0; mr-- {
				for col := 0; col < boardWidth; col++ {
					(*gameBoard)[mr][col] = (*gameBoard)[mr-1][col]
				}
			}
			for col := 0; col < boardWidth; col++ {
				(*gameBoard)[0][col] = 0
			}
			row++
		}
	}
	if cleared > 0 {
		linesCleared += cleared
		score += []int{0, 100, 300, 500, 800}[cleared]
		level = linesCleared/10 + 1
		// Acelera a gravidade conforme o nivel sobe
		tickInterval = 0.5 - float64(level-1)*0.04
		if tickInterval < 0.08 {
			tickInterval = 0.08
		}
	}
}

func applyGravity(gameBoard *[boardHeight][boardWidth]int) {
	if canPlace(currentPiece, currentX, currentY+1, &board) {
		currentY++
	} else {
		lockPiece(gameBoard)
		clearFullLines(gameBoard)
		spawnPiece()
	}
}

// ===== CONCEITO DA DISCIPLINA: PASSAGEM POR REFERENCIA =====
func gameTick(gameBoard *[boardHeight][boardWidth]int) {
	if gameOver {
		return
	}
	applyGravity(gameBoard)
}

// Update e chamado pelo Ebitengine a cada frame (60fps).
// Processa input e acumula tempo para o tick de gravidade.
func (g *Game) Update() error {
	if gameOver {
		if ebiten.IsKeyPressed(ebiten.KeyEnter) || ebiten.IsKeyPressed(ebiten.KeySpace) {
			initGame()
		}
		return nil
	}

	// Input — uma verificacao por frame, sem goroutines necessarias na GUI
	if inputJustPressed(ebiten.KeyA) || inputJustPressed(ebiten.KeyArrowLeft) {
		movePiece(-1, 0)
	}
	if inputJustPressed(ebiten.KeyD) || inputJustPressed(ebiten.KeyArrowRight) {
		movePiece(1, 0)
	}
	if inputJustPressed(ebiten.KeyS) || inputJustPressed(ebiten.KeyArrowDown) {
		movePiece(0, 1)
	}
	if inputJustPressed(ebiten.KeyW) || inputJustPressed(ebiten.KeyArrowUp) {
		rotatePiece()
	}
	if inputJustPressed(ebiten.KeySpace) {
		hardDrop()
	}

	// Gravidade baseada em tempo acumulado (delta time)
	tickAccum += 1.0 / 60.0
	if tickAccum >= tickInterval {
		tickAccum = 0
		gameTick(&board)
	}

	return nil
}

var prevKeys = map[ebiten.Key]bool{}

func inputJustPressed(key ebiten.Key) bool {
	pressed := ebiten.IsKeyPressed(key)
	wasPressed := prevKeys[key]
	prevKeys[key] = pressed
	return pressed && !wasPressed
}

// Draw e chamado pelo Ebitengine apos cada Update.
// Responsavel apenas pela renderizacao — sem logica de jogo aqui.
func (g *Game) Draw(screen *ebiten.Image) {
	// Fundo escuro
	screen.Fill(color.RGBA{18, 18, 24, 255})

	drawBoard(screen)
	drawCurrentPiece(screen)
	drawGhost(screen)
	drawSidePanel(screen)

	if gameOver {
		drawGameOver(screen)
	}
}

func boardOriginX() int { return padding }
func boardOriginY() int { return padding }

func drawBoard(screen *ebiten.Image) {
	ox := float32(boardOriginX())
	oy := float32(boardOriginY())
	bw := float32(boardWidth * cellSize)
	bh := float32(boardHeight * cellSize)

	// Fundo do tabuleiro
	vector.DrawFilledRect(screen, ox, oy, bw, bh, color.RGBA{26, 26, 36, 255}, false)

	// Grade
	for row := 0; row < boardHeight; row++ {
		for col := 0; col < boardWidth; col++ {
			x := ox + float32(col*cellSize)
			y := oy + float32(row*cellSize)
			c := board[row][col]
			if c != 0 {
				drawCell(screen, x, y, pieceColors[c], false)
			} else {
				// Linha de grade sutil
				vector.StrokeRect(screen, x+0.5, y+0.5, float32(cellSize)-1, float32(cellSize)-1, 0.5, color.RGBA{40, 40, 55, 255}, false)
			}
		}
	}

	// Borda do tabuleiro
	vector.StrokeRect(screen, ox-1, oy-1, bw+2, bh+2, 2, color.RGBA{80, 80, 120, 255}, false)
}

func drawCurrentPiece(screen *ebiten.Image) {
	ox := float32(boardOriginX())
	oy := float32(boardOriginY())
	for row := 0; row < pieceSize; row++ {
		for col := 0; col < pieceSize; col++ {
			if currentPiece[row][col] == 1 {
				x := ox + float32((currentX+col)*cellSize)
				y := oy + float32((currentY+row)*cellSize)
				drawCell(screen, x, y, pieceColors[currentColor], false)
			}
		}
	}
}

// Ghost: mostra onde a peca vai cair
func drawGhost(screen *ebiten.Image) {
	ghostY := currentY
	for canPlace(currentPiece, currentX, ghostY+1, &board) {
		ghostY++
	}
	if ghostY == currentY {
		return
	}
	ox := float32(boardOriginX())
	oy := float32(boardOriginY())
	gc := pieceColors[currentColor]
	ghost := color.RGBA{gc.R / 4, gc.G / 4, gc.B / 4, 180}
	for row := 0; row < pieceSize; row++ {
		for col := 0; col < pieceSize; col++ {
			if currentPiece[row][col] == 1 {
				x := ox + float32((currentX+col)*cellSize)
				y := oy + float32((ghostY+row)*cellSize)
				vector.DrawFilledRect(screen, x+1, y+1, float32(cellSize)-2, float32(cellSize)-2, ghost, false)
				vector.StrokeRect(screen, x+1, y+1, float32(cellSize)-2, float32(cellSize)-2, 1, color.RGBA{gc.R / 2, gc.G / 2, gc.B / 2, 200}, false)
			}
		}
	}
}

func drawCell(screen *ebiten.Image, x, y float32, c color.RGBA, small bool) {
	s := float32(cellSize)
	if small {
		s = 24
	}
	// Corpo principal
	vector.DrawFilledRect(screen, x+1, y+1, s-2, s-2, c, false)
	// Brilho superior
	bright := color.RGBA{
		clampAdd(c.R, 60), clampAdd(c.G, 60), clampAdd(c.B, 60), 200,
	}
	vector.DrawFilledRect(screen, x+1, y+1, s-2, 4, bright, false)
	vector.DrawFilledRect(screen, x+1, y+1, 4, s-2, bright, false)
	// Sombra inferior
	dark := color.RGBA{c.R / 2, c.G / 2, c.B / 2, 255}
	vector.DrawFilledRect(screen, x+1, y+s-5, s-2, 4, dark, false)
	vector.DrawFilledRect(screen, x+s-5, y+1, 4, s-2, dark, false)
}

func clampAdd(v uint8, add uint8) uint8 {
	if int(v)+int(add) > 255 {
		return 255
	}
	return v + add
}

func drawSidePanel(screen *ebiten.Image) {
	px := float32(boardOriginX() + boardWidth*cellSize + padding)
	py := float32(padding)

	ebitenutil.DebugPrintAt(screen, "TETRIS GO", int(px), int(py))
	ebitenutil.DebugPrintAt(screen, "Imperativo", int(px), int(py)+14)

	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Score\n%d", score), int(px), int(py)+46)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Nivel\n%d", level), int(px), int(py)+86)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Linhas\n%d", linesCleared), int(px), int(py)+126)

	ebitenutil.DebugPrintAt(screen, "Proxima", int(px), int(py)+166)
	drawNextPiece(screen, int(px), int(py)+182)

	ebitenutil.DebugPrintAt(screen, "Controles\nA/D mover\nW girar\nS descer\nESP drop", int(px), int(py)+290)
}

func drawNextPiece(screen *ebiten.Image, ox, oy int) {
	for row := 0; row < pieceSize; row++ {
		for col := 0; col < pieceSize; col++ {
			if nextPiece[row][col] == 1 {
				x := float32(ox + col*24)
				y := float32(oy + row*24)
				drawCell(screen, x, y, pieceColors[nextColor], true)
			}
		}
	}
}

func drawGameOver(screen *ebiten.Image) {
	ox := float32(boardOriginX())
	oy := float32(boardOriginY() + boardHeight*cellSize/2 - 30)
	bw := float32(boardWidth * cellSize)

	vector.DrawFilledRect(screen, ox, oy, bw, 60, color.RGBA{10, 10, 20, 220}, false)
	ebitenutil.DebugPrintAt(screen, "FIM DE JOGO", int(ox)+50, int(oy)+10)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Score: %d", score), int(ox)+65, int(oy)+26)
	ebitenutil.DebugPrintAt(screen, "Enter p/ reiniciar", int(ox)+30, int(oy)+42)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenW, screenH
}
