package main

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

type Player struct {
	X, Y                 float64
	Width, Height        float64
	VelocityX, VelocityY float64
	OnGround             bool
	Inventory            [9]InventoryItem
	SelectedSlot         int
	Mining               bool
	MiningTime           float64
	MiningX, MiningY     int
	PrevMouse            bool
	KeyStates            [9]bool
}

type InventoryItem struct {
	BlockType BlockType
	Count     int
}

func NewPlayer() *Player {
	p := &Player{
		X:            float64(screenWidth) / 2,
		Y:            float64(screenHeight) / 2,
		Width:        30,
		Height:       50,
		SelectedSlot: 0,
	}

	// дефолт инвентарь
	p.Inventory[0] = InventoryItem{Dirt, 64}
	p.Inventory[1] = InventoryItem{Stone, 64}
	p.Inventory[2] = InventoryItem{Wood, 64}

	return p
}

func (p *Player) Update(w *World) {
	// Гравитация
	p.VelocityY += 0.5
	if p.VelocityY > 15 {
		p.VelocityY = 15
	}

	// Трение
	p.VelocityX *= 0.9

	// управлеие
	speed := 1.0
	if ebiten.IsKeyPressed(ebiten.KeyA) {
		p.VelocityX -= speed
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) {
		p.VelocityX += speed
	}
	if (ebiten.IsKeyPressed(ebiten.KeySpace) || ebiten.IsKeyPressed(ebiten.KeyW)) && p.OnGround {
		p.VelocityY = -15
		p.OnGround = false
	}

	p.moveWithCollision(w)

	p.handleInventorySelection()

	p.handleMining(w)

	p.handleBlockPlacement(w)

	// камера
	w.CameraX = p.X - float64(screenWidth)/2 + p.Width/2
	w.CameraY = p.Y - float64(screenHeight)/2 + p.Height/2
}

func (p *Player) handleInventorySelection() {
	// 1-9
	keys := []ebiten.Key{
		ebiten.Key1, ebiten.Key2, ebiten.Key3, ebiten.Key4, ebiten.Key5,
		ebiten.Key6, ebiten.Key7, ebiten.Key8, ebiten.Key9,
	}

	for i, key := range keys {
		isPressed := ebiten.IsKeyPressed(key)
		if isPressed && !p.KeyStates[i] {
			p.SelectedSlot = i
		}
		p.KeyStates[i] = isPressed
	}
}

func (p *Player) handleMining(w *World) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("ОШИБКА В ДОБЫЧЕ: %v\n", r)
			p.resetMining()
		}
	}()

	// ЛКМ
	currentMouse := ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	if !currentMouse {
		p.resetMining()
		return
	}

	mouseX, mouseY := ebiten.CursorPosition()
	blockX := int((float64(mouseX) + w.CameraX) / blockSize)
	blockY := int((float64(mouseY) + w.CameraY) / blockSize)

	block := w.GetBlock(blockX, blockY)

	// проверка
	if block.Type == Air || !p.isBlockInMiningRange(blockX, blockY) {
		p.resetMining()
		return
	}

	if !p.Mining || p.MiningX != blockX || p.MiningY != blockY {
		p.Mining = true
		p.MiningTime = 0
		p.MiningX = blockX
		p.MiningY = blockY
	}

	p.MiningTime += 3.0 / 60.0

	const miningSpeed = 1.0
	hardness := math.Max(float64(block.Hardness), 1.0)
	if p.MiningTime >= hardness/miningSpeed {
		// если не воздух
		if block.Type != Air {
			p.addToInventory(block.Type)
			w.SetBlock(blockX, blockY, Block{Air, 0})
		}
		p.resetMining()
	}
}

func (p *Player) resetMining() {
	p.Mining = false
	p.MiningTime = 0
	p.MiningX, p.MiningY = 0, 0
}

func (p *Player) isBlockInMiningRange(blockX, blockY int) bool {
	playerBlockX := int(p.X / blockSize)
	playerBlockY := int(p.Y / blockSize)

	distance := math.Sqrt(
		math.Pow(float64(blockX-playerBlockX), 2) +
			math.Pow(float64(blockY-playerBlockY), 2),
	)

	return distance <= 6
}

func (p *Player) initOrContinueMining(blockX, blockY int) {
	if !p.Mining || p.MiningX != blockX || p.MiningY != blockY {
		p.Mining = true
		p.MiningTime = 0
		p.MiningX = blockX
		p.MiningY = blockY
	}
}

func (p *Player) isMiningComplete(block Block) bool {
	const miningSpeed = 1.0
	hardness := math.Max(float64(block.Hardness), 1.0)
	return p.MiningTime >= hardness/miningSpeed
}

func (p *Player) finishMining(w *World, blockX, blockY int, block Block) {
	defer func() {
		p.resetMining()
	}()

	if block.Type != Air {
		p.addToInventory(block.Type)
		w.SetBlock(blockX, blockY, Block{Air, 0})
	}
}
func (p *Player) handleBlockPlacement(w *World) {
	currentMouse := ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight)
	if currentMouse && !p.PrevMouse {
		mouseX, mouseY := ebiten.CursorPosition()
		blockX := int(float64(mouseX)+w.CameraX) / blockSize
		blockY := int(float64(mouseY)+w.CameraY) / blockSize

		if w.GetBlock(blockX, blockY).Type == Air && p.Inventory[p.SelectedSlot].Count > 0 {
			blockType := p.Inventory[p.SelectedSlot].BlockType
			if blockType != Air {
				w.SetBlock(blockX, blockY, Block{blockType, BlockHardness[blockType]})
				p.Inventory[p.SelectedSlot].Count--
			}
		}
	}
	p.PrevMouse = currentMouse
}

func (p *Player) moveWithCollision(w *World) {
	// пров по х
	p.X += p.VelocityX

	left := int(p.X / blockSize)
	right := int((p.X + p.Width - 1) / blockSize)
	top := int(p.Y / blockSize)
	bottom := int((p.Y + p.Height - 1) / blockSize)

	if p.VelocityX > 0 {
		for y := top; y <= bottom; y++ {
			if w.GetBlock(right, y).Type != Air {
				p.X = float64(right*blockSize - int(p.Width))
				p.VelocityX = 0
				break
			}
		}
	} else if p.VelocityX < 0 {
		for y := top; y <= bottom; y++ {
			if w.GetBlock(left, y).Type != Air {
				p.X = float64((left + 1) * blockSize)
				p.VelocityX = 0
				break
			}
		}
	}

	// Проверка по у
	p.OnGround = false
	p.Y += p.VelocityY

	left = int(p.X / blockSize)
	right = int((p.X + p.Width - 1) / blockSize)
	top = int(p.Y / blockSize)
	bottom = int((p.Y + p.Height - 1) / blockSize)

	if p.VelocityY > 0 {
		// Падаем
		for x := left; x <= right; x++ {
			if w.GetBlock(x, bottom).Type != Air {
				p.Y = float64(bottom*blockSize - int(p.Height))
				p.VelocityY = 0
				p.OnGround = true
				break
			}
		}
	} else if p.VelocityY < 0 {
		// Прыгаем
		for x := left; x <= right; x++ {
			if w.GetBlock(x, top).Type != Air {
				p.Y = float64((top + 1) * blockSize)
				p.VelocityY = 0
				break
			}
		}
	}
}

func (p *Player) addToInventory(blockType BlockType) {
	if blockType == Air {
		return
	}

	const maxStackSize = 64

	for i := range p.Inventory {
		if p.Inventory[i].BlockType == blockType && p.Inventory[i].Count < maxStackSize {
			p.Inventory[i].Count++
			return
		}
	}

	for i := range p.Inventory {
		if p.Inventory[i].Count == 0 {
			p.Inventory[i].BlockType = blockType
			p.Inventory[i].Count = 1
			return
		}
	}

	fmt.Println("Инвентарь полон, блок не добавлен:", blockType)
}

func (p *Player) Draw(screen *ebiten.Image) {
	playerImg := ebiten.NewImage(int(p.Width), int(p.Height))
	playerImg.Fill(color.RGBA{255, 0, 0, 255})

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(p.X-game.world.CameraX, p.Y-game.world.CameraY)
	screen.DrawImage(playerImg, op)

	p.drawInventory(screen)

	if p.Mining {
		defer func() {
			if r := recover(); r != nil {
				fmt.Println("Ошибка при отрисовке прогресса добычи:", r)
			}
		}()

		blockX, blockY := p.MiningX, p.MiningY
		block := game.world.GetBlock(blockX, blockY)

		if block.Type != Air {
			var progress float64
			if block.Hardness > 0 {
				progress = math.Min(1.0, p.MiningTime/float64(block.Hardness))
			} else {
				progress = 1.0
			}

			// рамка блока
			crackImg := ebiten.NewImage(blockSize, blockSize)
			crackImg.Fill(color.RGBA{255, 255, 255, 100})

			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(
				float64(blockX)*float64(blockSize)-game.world.CameraX,
				float64(blockY)*float64(blockSize)-game.world.CameraY,
			)
			screen.DrawImage(crackImg, op)

			// прогресс
			progressWidth := int(float64(blockSize) * progress)
			if progressWidth > 0 {
				progImg := ebiten.NewImage(progressWidth, 5)
				progImg.Fill(color.RGBA{255, 255, 0, 255})

				op = &ebiten.DrawImageOptions{}
				op.GeoM.Translate(
					float64(blockX)*float64(blockSize)-game.world.CameraX,
					float64(blockY)*float64(blockSize)-game.world.CameraY+float64(blockSize-5),
				)
				screen.DrawImage(progImg, op)
			}
		}
	}
}

func (p *Player) drawInventory(screen *ebiten.Image) {
	slotSize := 40
	padding := 4
	startX := (screenWidth - (slotSize+padding)*9 + padding) / 2
	startY := screenHeight - slotSize - 20

	bgImg := ebiten.NewImage((slotSize+padding)*9, slotSize+padding*2)
	bgImg.Fill(color.RGBA{50, 50, 50, 200})

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(startX-padding), float64(startY-padding))
	screen.DrawImage(bgImg, op)

	for i := 0; i < 9; i++ {
		x := startX + i*(slotSize+padding)

		slotImg := ebiten.NewImage(slotSize, slotSize)
		if i == p.SelectedSlot {
			slotImg.Fill(color.RGBA{100, 100, 255, 255})
		} else {
			slotImg.Fill(color.RGBA{70, 70, 70, 255})
		}

		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(x), float64(startY))
		screen.DrawImage(slotImg, op)

		if p.Inventory[i].Count > 0 {
			blockImg := ebiten.NewImage(slotSize-10, slotSize-10)
			blockImg.Fill(BlockColors[p.Inventory[i].BlockType])

			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(x+5), float64(startY+5))
			screen.DrawImage(blockImg, op)
		}
	}
}
