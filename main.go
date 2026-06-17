package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

//go:embed gameover.png
var gopherBytes []byte

//go:embed start.png
var startBytes []byte

//go:embed font.ttf
var fontRegularBytes []byte

//go:embed font_bold.ttf
var fontBoldBytes []byte

//go:embed bgm.mp3
var bgmBytes []byte

//go:embed drop.mp3
var dropBytes []byte

//go:embed line.mp3
var lineBytes []byte

//go:embed gover.mp3
var goverBytes []byte

//go:embed loser.mp3
var loserBytes []byte

const boardHeight = 18
const boardWidth = 14
const pieceSize = 4
const cellSize = 36
const padding = 16
const sidePanel = 270

const screenW = boardWidth*cellSize + padding*2 + sidePanel
const screenH = boardHeight*cellSize + padding*2

// ===== ESTADO GLOBAL (paradigma imperativo) =====
var board [boardHeight][boardWidth]int

var currentPiece [pieceSize][pieceSize]int
var currentColor int
var currentX int
var currentY int
var score int
var highScore int
var level int
var linesCleared int
var gameOver bool
var gameStarted bool
var seed int
var tickAccum float64
var tickInterval float64

// Flash de linhas
var flashLines [boardHeight]bool
var flashTimer float64

const flashDuration = 0.18

// Animacao de game over
var gameOverTimer float64

const gameOverAnimDuration = 1.5

var gopherImage *ebiten.Image
var startImage *ebiten.Image

// ===== ESTADO GLOBAL DE AUDIO (paradigma imperativo: players sao reutilizados,
// nunca recriados; controle de "tocar so uma vez" feito via flags globais) =====
var audioContext *audio.Context
var bgmPlayer *audio.Player
var dropPlayer *audio.Player
var linePlayer *audio.Player
var goverPlayer *audio.Player
var loserPlayer *audio.Player

// goverPlayed garante que gover.mp3 toque uma unica vez, exatamente no
// instante em que a imagem de game over aparece na tela.
var goverPlayed bool

// blinkTimer controla o pisca-pisca do prompt
var blinkTimer float64

// Fontes TTF
var faceRegularSm *text.GoTextFace
var faceRegularMd *text.GoTextFace
var faceBoldSm *text.GoTextFace
var faceBoldMd *text.GoTextFace
var faceBoldLg *text.GoTextFace
var faceBoldXl *text.GoTextFace

var pieces = [7][pieceSize][pieceSize]int{
	{{0, 0, 0, 0}, {1, 1, 1, 1}, {0, 0, 0, 0}, {0, 0, 0, 0}},
	{{1, 1, 0, 0}, {1, 1, 0, 0}, {0, 0, 0, 0}, {0, 0, 0, 0}},
	{{0, 1, 0, 0}, {1, 1, 1, 0}, {0, 0, 0, 0}, {0, 0, 0, 0}},
	{{0, 1, 1, 0}, {1, 1, 0, 0}, {0, 0, 0, 0}, {0, 0, 0, 0}},
	{{1, 1, 0, 0}, {0, 1, 1, 0}, {0, 0, 0, 0}, {0, 0, 0, 0}},
	{{1, 0, 0, 0}, {1, 1, 1, 0}, {0, 0, 0, 0}, {0, 0, 0, 0}},
	{{0, 0, 1, 0}, {1, 1, 1, 0}, {0, 0, 0, 0}, {0, 0, 0, 0}},
}

// Fila das proximas 3 pecas
const previewCount = 3

var nextPieces [previewCount][pieceSize][pieceSize]int
var nextColors [previewCount]int

var pieceColors = [8]color.RGBA{
	{0, 0, 0, 0},
	{0, 188, 212, 255},
	{255, 193, 7, 255},
	{156, 39, 176, 255},
	{76, 175, 80, 255},
	{244, 67, 54, 255},
	{33, 150, 243, 255},
	{158, 158, 158, 255},
}

type Game struct{}

func main() {
	ebiten.SetWindowSize(screenW, screenH)
	ebiten.SetWindowTitle("Tetris Imperativo — Go")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeDisabled)

	// Carrega fontes TTF
	loadFontsFromBytes()

	// Carrega e prepara os players de audio
	loadAudio()

	img, _, err := ebitenutil.NewImageFromReader(bytes.NewReader(gopherBytes))
	if err == nil {
		gopherImage = img
	}

	simg, _, serr := ebitenutil.NewImageFromReader(bytes.NewReader(startBytes))
	if serr == nil {
		startImage = simg
	}

	highScore = 0
	gameStarted = false
	gameOver = false

	if err := ebiten.RunGame(&Game{}); err != nil {
		fmt.Println(err)
	}
}

func loadFontsFromBytes() {
	makeSource := func(data []byte) *text.GoTextFaceSource {
		src, err := text.NewGoTextFaceSource(bytes.NewReader(data))
		if err != nil {
			panic(fmt.Sprintf("falha ao carregar fonte: %v", err))
		}
		return src
	}

	regSrc := makeSource(fontRegularBytes)
	boldSrc := makeSource(fontBoldBytes)

	faceRegularSm = &text.GoTextFace{Source: regSrc, Size: 11}
	faceRegularMd = &text.GoTextFace{Source: regSrc, Size: 13}
	faceBoldSm = &text.GoTextFace{Source: boldSrc, Size: 11}
	faceBoldMd = &text.GoTextFace{Source: boldSrc, Size: 13}
	faceBoldLg = &text.GoTextFace{Source: boldSrc, Size: 18}
	faceBoldXl = &text.GoTextFace{Source: boldSrc, Size: 26}
}

// ===== CONCEITO DA DISCIPLINA: ESTADO GLOBAL + MUTACAO EXPLICITA =====
// loadAudio decodifica cada mp3 uma unica vez e guarda o player resultante
// em variavel global. Os players sao reaproveitados (Rewind/Seek) em vez de
// recriados a cada som, evitando overhead e mantendo o estilo imperativo.
func loadAudio() {
	audioContext = audio.NewContext(48000)

	makePlayer := func(data []byte) *audio.Player {
		stream, err := mp3.DecodeWithSampleRate(48000, bytes.NewReader(data))
		if err != nil {
			panic(fmt.Sprintf("falha ao decodificar mp3: %v", err))
		}
		player, err := audioContext.NewPlayer(stream)
		if err != nil {
			panic(fmt.Sprintf("falha ao criar player de audio: %v", err))
		}
		return player
	}

	bgmPlayer = makePlayer(bgmBytes)
	dropPlayer = makePlayer(dropBytes)
	linePlayer = makePlayer(lineBytes)
	goverPlayer = makePlayer(goverBytes)
	loserPlayer = makePlayer(loserBytes)
}

// playOneShot reinicia o player do zero e toca. Usado para efeitos sonoros
// curtos (drop, line, gover, loser) que podem repetir a qualquer momento.
func playOneShot(p *audio.Player) {
	if p == nil {
		return
	}
	p.Pause()
	if err := p.Rewind(); err != nil {
		return
	}
	p.Play()
}

// restartBGM para a musica de fundo (se estiver tocando) e a reinicia do
// zero (posicao 0), exatamente como pedido: toda partida nova comeca a bgm
// do absoluto inicio, nunca de onde parou.
func restartBGM() {
	if bgmPlayer == nil {
		return
	}
	bgmPlayer.Pause()
	bgmPlayer.Rewind()
	bgmPlayer.Play()
}

// stopBGM interrompe a musica de fundo (usado no instante do game over).
func stopBGM() {
	if bgmPlayer == nil {
		return
	}
	bgmPlayer.Pause()
}

// loopBGM verifica se a bgm terminou e a reinicia, criando um loop manual.
// Chamado a cada frame em Update enquanto a partida estiver em andamento.
func loopBGM() {
	if bgmPlayer == nil || !gameStarted || gameOver {
		return
	}
	if !bgmPlayer.IsPlaying() {
		bgmPlayer.Rewind()
		bgmPlayer.Play()
	}
}

// drawText desenha texto com a face especificada, cor e posicao (x, y = topo-esquerdo).
func drawText(screen *ebiten.Image, str string, face *text.GoTextFace, x, y int, clr color.RGBA) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	op.ColorScale.ScaleWithColor(clr)
	text.Draw(screen, str, face, op)
}

func initGame() {
	for row := 0; row < boardHeight; row++ {
		for col := 0; col < boardWidth; col++ {
			board[row][col] = 0
		}
	}
	for row := 0; row < boardHeight; row++ {
		flashLines[row] = false
	}

	score = 0
	level = 1
	linesCleared = 0
	gameOver = false
	gameStarted = true
	gameOverTimer = 0
	tickAccum = 0
	tickInterval = 0.5
	flashTimer = 0
	seed = int(time.Now().UnixNano() % 100000)

	// Audio: reseta flag de gover e reinicia a musica de fundo do zero
	goverPlayed = false
	restartBGM()

	for i := 0; i < previewCount; i++ {
		ni := nextPieceIndex()
		for r := 0; r < pieceSize; r++ {
			for c := 0; c < pieceSize; c++ {
				nextPieces[i][r][c] = pieces[ni][r][c]
			}
		}
		nextColors[i] = ni + 1
	}

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
			currentPiece[r][c] = nextPieces[0][r][c]
		}
	}
	currentColor = nextColors[0]

	for i := 0; i < previewCount-1; i++ {
		for r := 0; r < pieceSize; r++ {
			for c := 0; c < pieceSize; c++ {
				nextPieces[i][r][c] = nextPieces[i+1][r][c]
			}
		}
		nextColors[i] = nextColors[i+1]
	}

	ni := nextPieceIndex()
	for r := 0; r < pieceSize; r++ {
		for c := 0; c < pieceSize; c++ {
			nextPieces[previewCount-1][r][c] = pieces[ni][r][c]
		}
	}
	nextColors[previewCount-1] = ni + 1

	currentX = boardWidth/2 - 2
	currentY = 0

	if !canPlace(currentPiece, currentX, currentY, &board) {
		gameOver = true
		gameOverTimer = gameOverAnimDuration
		if score > highScore {
			highScore = score
		}
		// O jogador perdeu agora mesmo: para a bgm e toca o som de derrota
		// na hora exata da perda (nao espera a animacao/imagem de game over).
		stopBGM()
		playOneShot(loserPlayer)
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
	dropped := 0
	for canPlace(currentPiece, currentX, currentY+1, &board) {
		currentY++
		dropped++
	}
	// Hard drop: 2 pontos por linha caida
	score += dropped * 2
	if score > highScore {
		highScore = score
	}
	lockPiece(&board)
	markFullLines(&board)
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
	playOneShot(dropPlayer)
}

func markFullLines(gameBoard *[boardHeight][boardWidth]int) {
	found := false
	for row := 0; row < boardHeight; row++ {
		full := true
		for col := 0; col < boardWidth; col++ {
			if (*gameBoard)[row][col] == 0 {
				full = false
				break
			}
		}
		if full {
			flashLines[row] = true
			found = true
		}
	}
	if found {
		flashTimer = flashDuration
		// Toca o som de linha completa uma unica vez, mesmo que 2, 3 ou 4
		// linhas tenham sido completadas simultaneamente neste lock.
		playOneShot(linePlayer)
	} else {
		spawnPiece()
	}
}

func clearFullLines(gameBoard *[boardHeight][boardWidth]int) {
	cleared := 0
	for row := boardHeight - 1; row >= 0; row-- {
		if flashLines[row] {
			cleared++
			for mr := row; mr > 0; mr-- {
				for col := 0; col < boardWidth; col++ {
					(*gameBoard)[mr][col] = (*gameBoard)[mr-1][col]
				}
				flashLines[mr] = flashLines[mr-1]
			}
			for col := 0; col < boardWidth; col++ {
				(*gameBoard)[0][col] = 0
			}
			flashLines[0] = false
			row++
		}
	}
	if cleared > 0 {
		linesCleared += cleared
		// ===== SISTEMA DE PONTUACAO: bonus por combo + multiplicador de nivel =====
		basePoints := []int{0, 100, 300, 500, 800}[cleared]
		score += basePoints * level
		if score > highScore {
			highScore = score
		}
		level = linesCleared/10 + 1
		tickInterval = 0.5 - float64(level-1)*0.04
		if tickInterval < 0.08 {
			tickInterval = 0.08
		}
	}
	spawnPiece()
}

func applyGravity(gameBoard *[boardHeight][boardWidth]int) {
	if canPlace(currentPiece, currentX, currentY+1, &board) {
		currentY++
	} else {
		lockPiece(gameBoard)
		markFullLines(gameBoard)
	}
}

// ===== CONCEITO DA DISCIPLINA: PASSAGEM POR REFERENCIA =====
func gameTick(gameBoard *[boardHeight][boardWidth]int) {
	if gameOver {
		return
	}
	applyGravity(gameBoard)
}

var prevKeys = map[ebiten.Key]bool{}

func inputJustPressed(key ebiten.Key) bool {
	pressed := ebiten.IsKeyPressed(key)
	wasPressed := prevKeys[key]
	prevKeys[key] = pressed
	return pressed && !wasPressed
}

func (g *Game) Update() error {
	if !gameStarted {
		blinkTimer += 1.0 / 60.0
		if inputJustPressed(ebiten.KeyEnter) || inputJustPressed(ebiten.KeySpace) {
			initGame()
		}
		return nil
	}

	if gameOver {
		blinkTimer += 1.0 / 60.0
		if gameOverTimer > 0 {
			gameOverTimer -= 1.0 / 60.0
		}
		if gameOverTimer <= 0 {
			// Dispara gover.mp3 exatamente no frame em que a animacao termina
			// e a imagem de game over passa a ser exibida na tela.
			if !goverPlayed {
				goverPlayed = true
				playOneShot(goverPlayer)
			}
			if inputJustPressed(ebiten.KeyEnter) || inputJustPressed(ebiten.KeySpace) {
				gameStarted = false
			}
		}
		return nil
	}

	if flashTimer > 0 {
		flashTimer -= 1.0 / 60.0
		loopBGM()
		if flashTimer <= 0 {
			flashTimer = 0
			clearFullLines(&board)
		}
		return nil
	}

	if inputJustPressed(ebiten.KeyA) || inputJustPressed(ebiten.KeyArrowLeft) {
		movePiece(-1, 0)
	}
	if inputJustPressed(ebiten.KeyD) || inputJustPressed(ebiten.KeyArrowRight) {
		movePiece(1, 0)
	}
	// ===== SOFT DROP: S/seta-baixo soma 1 ponto por linha descida manualmente =====
	if inputJustPressed(ebiten.KeyS) || inputJustPressed(ebiten.KeyArrowDown) {
		if canPlace(currentPiece, currentX, currentY+1, &board) {
			currentY++
			score += 1
			if score > highScore {
				highScore = score
			}
		}
	}
	if inputJustPressed(ebiten.KeyW) || inputJustPressed(ebiten.KeyArrowUp) {
		rotatePiece()
	}
	if inputJustPressed(ebiten.KeySpace) {
		hardDrop()
	}

	blinkTimer += 1.0 / 60.0

	// Garante que a bgm continue tocando em loop enquanto a partida estiver ativa
	loopBGM()

	tickAccum += 1.0 / 60.0
	if tickAccum >= tickInterval {
		tickAccum = 0
		gameTick(&board)
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{18, 18, 24, 255})

	if !gameStarted {
		drawStartScreen(screen)
		return
	}

	drawBoard(screen)
	if flashTimer <= 0 && !gameOver {
		drawCurrentPiece(screen)
		drawGhost(screen)
	}
	drawSidePanel(screen)

	if gameOver {
		drawGameOver(screen)
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenW, screenH
}

func drawStartScreen(screen *ebiten.Image) {
	screen.Fill(color.RGBA{0, 0, 0, 255})
	if startImage != nil {
		iw := startImage.Bounds().Dx()
		ih := startImage.Bounds().Dy()
		// Escala "contain": a imagem inteira sempre cabe na tela, sem cortar bordas.
		scaleX := float64(screenW) / float64(iw)
		scaleY := float64(screenH) / float64(ih)
		scale := scaleX
		if scaleY < scale {
			scale = scaleY
		}
		drawW := float64(iw) * scale
		drawH := float64(ih) * scale
		drawX := (float64(screenW) - drawW) / 2
		drawY := (float64(screenH) - drawH) / 2
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(scale, scale)
		op.GeoM.Translate(drawX, drawY)
		screen.DrawImage(startImage, op)
	}
	// Prompt em texto, numa faixa propria na base da tela
	drawPrompt(screen)
}

func boardOriginX() int { return padding }
func boardOriginY() int { return padding }

func drawBoard(screen *ebiten.Image) {
	ox := float32(boardOriginX())
	oy := float32(boardOriginY())
	bw := float32(boardWidth * cellSize)
	bh := float32(boardHeight * cellSize)

	vector.DrawFilledRect(screen, ox, oy, bw, bh, color.RGBA{26, 26, 36, 255}, false)

	flashOn := int(flashTimer*20)%2 == 0

	for row := 0; row < boardHeight; row++ {
		for col := 0; col < boardWidth; col++ {
			x := ox + float32(col*cellSize)
			y := oy + float32(row*cellSize)
			c := board[row][col]

			if flashLines[row] {
				if flashOn {
					drawCell(screen, x, y, color.RGBA{255, 255, 255, 255}, false)
				} else if c != 0 {
					drawCell(screen, x, y, pieceColors[c], false)
				}
			} else if c != 0 {
				drawCell(screen, x, y, pieceColors[c], false)
			} else {
				vector.StrokeRect(screen, x+0.5, y+0.5, float32(cellSize)-1, float32(cellSize)-1, 0.5, color.RGBA{40, 40, 55, 255}, false)
			}
		}
	}

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
		s = 22
	}
	vector.DrawFilledRect(screen, x+1, y+1, s-2, s-2, c, false)
	bright := color.RGBA{clampAdd(c.R, 60), clampAdd(c.G, 60), clampAdd(c.B, 60), 200}
	vector.DrawFilledRect(screen, x+1, y+1, s-2, 4, bright, false)
	vector.DrawFilledRect(screen, x+1, y+1, 4, s-2, bright, false)
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

// Paleta
var panelBg = color.RGBA{14, 16, 30, 255}
var cardBg = color.RGBA{24, 27, 48, 255}
var accentCyan = color.RGBA{56, 217, 235, 255}
var accentPurple = color.RGBA{156, 100, 230, 255}
var textWhite = color.RGBA{230, 232, 255, 255}
var textDim = color.RGBA{140, 145, 170, 255}

// drawPrompt desenha o texto "ESPAÇO OU ENTER PARA JOGAR" com efeito de pisca suave,
// sempre numa faixa escura na base da tela para garantir legibilidade sobre qualquer imagem.
func drawPrompt(screen *ebiten.Image) {
	promptStr := "[ ESPAÇO ou ENTER para jogar ]"

	// Opacidade oscila entre 40% e 100% usando seno
	alpha := float32(0.5 + 0.5*math.Sin(blinkTimer*3.5))
	if alpha < 0.4 {
		alpha = 0.4
	}

	barH := float32(40)
	barY := float32(screenH) - barH
	vector.DrawFilledRect(screen, 0, barY, float32(screenW), barH, color.RGBA{0, 0, 0, 200}, false)

	sw, sh := text.Measure(promptStr, faceBoldMd, 0)
	tx := screenW/2 - int(sw)/2
	ty := int(barY) + int(float64(barH)-sh)/2

	clr := accentCyan
	clr.A = uint8(255 * alpha)
	drawText(screen, promptStr, faceBoldMd, tx, ty, clr)
}

func drawSidePanel(screen *ebiten.Image) {
	px := float32(boardOriginX() + boardWidth*cellSize + padding)
	py := float32(padding)
	panelW := float32(sidePanel - padding)

	// Fundo geral do painel
	vector.DrawFilledRect(screen, px-6, py-6, panelW+6, float32(screenH-padding*2+12), panelBg, false)

	cursorY := py

	// ===== Card de pontuacao =====
	scoreCardH := float32(100)
	drawCard(screen, px-2, cursorY-2, panelW, scoreCardH)
	drawText(screen, "PONTUAÇÃO", faceBoldSm, int(px)+10, int(cursorY)+6, accentCyan)
	drawText(screen, fmt.Sprintf("%d", score), faceBoldXl, int(px)+10, int(cursorY)+22, textWhite)
	drawDivider(screen, px+10, cursorY+54, panelW-20)
	drawText(screen, fmt.Sprintf("Recorde: %d", highScore), faceRegularMd, int(px)+10, int(cursorY)+60, textDim)
	drawText(screen, fmt.Sprintf("Nível %d   |   %d linhas", level, linesCleared), faceRegularSm, int(px)+10, int(cursorY)+78, textDim)
	cursorY += scoreCardH + 8

	// ===== Card de proximas pecas =====
	previewCardH := float32(186)
	drawCard(screen, px-2, cursorY-2, panelW, previewCardH)
	drawText(screen, "PRÓXIMAS", faceBoldSm, int(px)+10, int(cursorY)+6, accentCyan)
	drawDivider(screen, px+10, cursorY+24, panelW-20)
	drawPreviewPieces(screen, int(px)+10, int(cursorY)+32)
	cursorY += previewCardH + 8

	// ===== Card de controles =====
	controlsCardH := float32(136)
	drawCard(screen, px-2, cursorY-2, panelW, controlsCardH)
	drawText(screen, "CONTROLES", faceBoldSm, int(px)+10, int(cursorY)+6, accentCyan)
	drawDivider(screen, px+10, cursorY+24, panelW-20)

	type ctrl struct{ key, desc string }
	controles := []ctrl{
		{"A / D  ou  ←→", "mover"},
		{"W  ou  ↑", "girar"},
		{"S  ou  ↓", "descer  (+1 pt)"},
		{"ESPAÇO", "hard drop  (+2 pt/linha)"},
	}
	for i, c := range controles {
		y := int(cursorY) + 34 + i*24
		drawText(screen, c.key, faceBoldSm, int(px)+10, y, textWhite)
		drawText(screen, c.desc, faceRegularSm, int(px)+10, y+13, textDim)
	}
}

func drawCard(screen *ebiten.Image, x, y, w, h float32) {
	vector.DrawFilledRect(screen, x, y, w, h, cardBg, false)
	vector.StrokeRect(screen, x, y, w, h, 1, color.RGBA{50, 55, 80, 255}, false)
}

func drawDivider(screen *ebiten.Image, x, y, w float32) {
	vector.StrokeLine(screen, x, y, x+w, y, 1, color.RGBA{45, 50, 75, 255}, false)
}

func drawPreviewPieces(screen *ebiten.Image, ox, oy int) {
	slotH := 48
	previewCell := float32(16)
	highlightW := float32(sidePanel - padding*2 - 8)

	for i := 0; i < previewCount; i++ {
		alpha := uint8(255 - i*70)
		slotY := oy + i*slotH

		if i == 0 {
			vector.StrokeRect(screen, float32(ox)-4, float32(slotY)-4, highlightW, float32(slotH-6), 1, accentCyan, false)
		}

		for row := 0; row < pieceSize; row++ {
			for col := 0; col < pieceSize; col++ {
				if nextPieces[i][row][col] == 1 {
					c := pieceColors[nextColors[i]]
					c.A = alpha
					x := float32(ox+6) + float32(col)*previewCell
					y := float32(slotY) + float32(row)*(previewCell*0.6)
					drawMiniCell(screen, x, y, c, previewCell)
				}
			}
		}
	}
}

func drawMiniCell(screen *ebiten.Image, x, y float32, c color.RGBA, s float32) {
	vector.DrawFilledRect(screen, x+1, y+1, s-2, s-2, c, false)
	bright := color.RGBA{clampAdd(c.R, 60), clampAdd(c.G, 60), clampAdd(c.B, 60), 200}
	vector.DrawFilledRect(screen, x+1, y+1, s-2, 3, bright, false)
	dark := color.RGBA{c.R / 2, c.G / 2, c.B / 2, 255}
	vector.DrawFilledRect(screen, x+1, y+s-4, s-2, 3, dark, false)
}

// ===== ANIMACAO DE GAME OVER =====
func drawGameOver(screen *ebiten.Image) {
	progress := 1.0 - (gameOverTimer / gameOverAnimDuration)
	if progress < 0 {
		progress = 0
	}

	linesFallen := int(math.Round(progress * float64(boardHeight)))

	ox := float32(boardOriginX())
	oy := float32(boardOriginY())

	for row := 0; row < linesFallen && row < boardHeight; row++ {
		y := oy + float32(row*cellSize)
		vector.DrawFilledRect(screen, ox, y, float32(boardWidth*cellSize), float32(cellSize), color.RGBA{40, 40, 50, 220}, false)
	}

	if gameOverTimer <= 0 {
		// Fundo escuro cobrindo toda a tela, ja que a imagem nao preenche mais 100%
		vector.DrawFilledRect(screen, 0, 0, float32(screenW), float32(screenH), color.RGBA{10, 10, 18, 235}, false)

		// Imagem inteira, sem cortar, com margem e respeitando espaco da faixa de score
		if gopherImage != nil {
			iw := gopherImage.Bounds().Dx()
			ih := gopherImage.Bounds().Dy()
			margin := 40.0
			scoreBarSpace := 70.0
			availW := float64(screenW) - margin*2
			availH := float64(screenH) - margin*2 - scoreBarSpace
			scaleX := availW / float64(iw)
			scaleY := availH / float64(ih)
			scale := scaleX
			if scaleY < scale {
				scale = scaleY
			}
			drawW := float64(iw) * scale
			drawH := float64(ih) * scale
			drawX := (float64(screenW) - drawW) / 2
			drawY := (float64(screenH) - scoreBarSpace - drawH) / 2
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(scale, scale)
			op.GeoM.Translate(drawX, drawY)
			screen.DrawImage(gopherImage, op)
		}

		// Overlay escuro na faixa inferior, atras do score e do prompt
		scoreBarH := float32(70)
		vector.DrawFilledRect(screen, 0, float32(screenH)-scoreBarH, float32(screenW), scoreBarH, color.RGBA{0, 0, 0, 170}, false)

		// Score centralizado dentro dessa faixa, acima do prompt
		scoreStr := fmt.Sprintf("Score: %d   |   Recorde: %d", score, highScore)
		sw, _ := text.Measure(scoreStr, faceBoldMd, 0)
		scoreY := int(float64(screenH) - float64(scoreBarH) + 12)
		drawText(screen, scoreStr, faceBoldMd, screenW/2-int(sw)/2, scoreY, textWhite)

		drawPrompt(screen)
	}
}
