package main

import (
	"fmt"
	"image/color"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

const (
	screenWidth  = 1280
	screenHeight = 720
	blockSize    = 40
	chunkSize    = 16
)

var (
	game *Game
)

type Game struct {
	world     *World
	player    *Player
	debugMode bool
	prevF3    bool
}

func (g *Game) Update() error {
	g.player.Update(g.world)

	// vключение/vыключение режима отладки
	f3Pressed := ebiten.IsKeyPressed(ebiten.KeyF3)
	if f3Pressed && !g.prevF3 {
		g.debugMode = !g.debugMode
	}
	g.prevF3 = f3Pressed

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	// fon nebo
	screen.Fill(color.RGBA{135, 206, 235, 255})

	// otrisovka
	g.world.Draw(screen)
	g.player.Draw(screen)

	mouseX, mouseY := ebiten.CursorPosition()
	blockX := int(float64(mouseX)+g.world.CameraX) / blockSize
	blockY := int(float64(mouseY)+g.world.CameraY) / blockSize

	targetImg := ebiten.NewImage(blockSize, blockSize)
	targetImg.Fill(color.RGBA{255, 255, 255, 100})

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(blockX)*float64(blockSize)-g.world.CameraX,
		float64(blockY)*float64(blockSize)-g.world.CameraY)
	screen.DrawImage(targetImg, op)

	// info
	if g.debugMode {
		playerBlockX := int(g.player.X) / blockSize
		playerBlockY := int(g.player.Y) / blockSize

		fps := ebiten.CurrentFPS()
		debug := fmt.Sprintf("FPS: %.2f\nX: %.2f, Y: %.2f\nBlock: %d, %d\nVelocity: %.2f, %.2f",
			fps, g.player.X, g.player.Y, playerBlockX, playerBlockY,
			g.player.VelocityX, g.player.VelocityY)

		ebitenutil.DebugPrint(screen, debug)
	}

	// instr
	instructions := "WASD/Пробел: Движение | ЛКМ: Добыча | ПКМ: Размещение | 1-9: Выбор блока | F3: Отладка"
	ebitenutil.DebugPrintAt(screen, instructions, 10, 10)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	game = &Game{
		world:  NewWorld(),
		player: NewPlayer(),
	}

	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Майнкрафт")
	ebiten.SetWindowResizable(true)

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
