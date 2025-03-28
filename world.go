package main

import (
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
)

type BlockType int

const (
	Air BlockType = iota
	Grass
	Dirt
	Stone
	Wood
	Leaves
)

var BlockColors = map[BlockType]color.RGBA{
	Air:    {0, 0, 0, 0},
	Grass:  {85, 170, 0, 255},
	Dirt:   {139, 69, 19, 255},
	Stone:  {128, 128, 128, 255},
	Wood:   {160, 82, 45, 255},
	Leaves: {34, 139, 34, 255},
}

var BlockHardness = map[BlockType]uint8{
	Air:    0,
	Grass:  1,
	Dirt:   2,
	Stone:  5,
	Wood:   3,
	Leaves: 1,
}

type Block struct {
	Type     BlockType
	Hardness uint8
}

type Chunk struct {
	Blocks [chunkSize][chunkSize]Block
}

type World struct {
	Chunks           map[[2]int]*Chunk
	Seed             int64
	CameraX, CameraY float64
}

func NewWorld() *World {
	w := &World{
		Chunks: make(map[[2]int]*Chunk),
		Seed:   rand.Int63(),
	}
	w.Generate()
	return w
}

func (w *World) noise(x, y float64) float64 {
	nx := math.Sin(x*0.1+float64(w.Seed))*0.5 + 0.5
	ny := math.Sin(y*0.1+float64(w.Seed)*0.7)*0.5 + 0.5
	return nx*0.7 + ny*0.3
}

func (w *World) Generate() {
	// генерируем мир
	for cx := -5; cx <= 5; cx++ {
		for cy := -2; cy <= 3; cy++ {
			chunk := &Chunk{}
			for x := 0; x < chunkSize; x++ {
				for y := 0; y < chunkSize; y++ {
					worldX := cx*chunkSize + x
					worldY := cy*chunkSize + y

					height := int(10 + w.noise(float64(worldX), 0)*5)

					caveNoise := w.noise(float64(worldX)*0.15, float64(worldY)*0.15)

					if worldY > height+5 && caveNoise > 0.6 {
						// Пещера
						chunk.Blocks[x][y] = Block{Air, 0}
					} else if worldY > height {
						// Камень с разной твердостью
						chunk.Blocks[x][y] = Block{Stone, BlockHardness[Stone]}
					} else if worldY > height-3 {
						// Слой земли
						chunk.Blocks[x][y] = Block{Dirt, BlockHardness[Dirt]}
					} else if worldY == height-3 {
						// Трава
						chunk.Blocks[x][y] = Block{Grass, BlockHardness[Grass]}
					} else {
						// Воздух
						chunk.Blocks[x][y] = Block{Air, 0}
					}

					// Генерация деревьев
					if worldY == height-3 && rand.Float64() < 0.1 && x > 2 && x < chunkSize-2 {
						// Ствол дерева
						treeHeight := 4 + rand.Intn(3)
						for th := 1; th <= treeHeight; th++ {
							if y-th >= 0 {
								chunk.Blocks[x][y-th] = Block{Wood, BlockHardness[Wood]}
							}
						}

						// Листья
						for lx := -2; lx <= 2; lx++ {
							for ly := -3; ly <= 0; ly++ {
								nx, ny := x+lx, y-treeHeight+ly-1
								if nx >= 0 && nx < chunkSize && ny >= 0 && ny < chunkSize {
									if chunk.Blocks[nx][ny].Type == Air {
										chunk.Blocks[nx][ny] = Block{Leaves, BlockHardness[Leaves]}
									}
								}
							}
						}
					}
				}
			}
			w.Chunks[[2]int{cx, cy}] = chunk
		}
	}
}

func (w *World) GetBlock(x, y int) Block {
	if y < -100 || y > 100 {
		return Block{Air, 0}
	}

	chunkX := x / chunkSize
	chunkY := y / chunkSize

	if x < 0 && x%chunkSize != 0 {
		chunkX--
	}
	if y < 0 && y%chunkSize != 0 {
		chunkY--
	}

	inChunkX := (x%chunkSize + chunkSize) % chunkSize
	inChunkY := (y%chunkSize + chunkSize) % chunkSize

	chunk, exists := w.Chunks[[2]int{chunkX, chunkY}]
	if !exists {
		return Block{Air, 0}
	}

	return chunk.Blocks[inChunkX][inChunkY]
}

func (w *World) SetBlock(x, y int, block Block) {
	chunkX := x / chunkSize
	chunkY := y / chunkSize

	if x < 0 {
		chunkX = (x - chunkSize + 1) / chunkSize
	}
	if y < 0 {
		chunkY = (y - chunkSize + 1) / chunkSize
	}

	inChunkX := (x%chunkSize + chunkSize) % chunkSize
	inChunkY := (y%chunkSize + chunkSize) % chunkSize

	if inChunkX < 0 || inChunkX >= chunkSize || inChunkY < 0 || inChunkY >= chunkSize {
		return
	}

	chunk, exists := w.Chunks[[2]int{chunkX, chunkY}]
	if !exists {
		return
	}

	chunk.Blocks[inChunkX][inChunkY] = block
}

func (w *World) Draw(screen *ebiten.Image) {
	visibleWidth := screenWidth / blockSize
	visibleHeight := screenHeight / blockSize

	minX := int(w.CameraX/blockSize) - 1
	minY := int(w.CameraY/blockSize) - 1
	maxX := minX + visibleWidth + 2
	maxY := minY + visibleHeight + 2

	minChunkX := minX / chunkSize
	minChunkY := minY / chunkSize
	maxChunkX := maxX / chunkSize
	maxChunkY := maxY / chunkSize

	for cx := minChunkX; cx <= maxChunkX; cx++ {
		for cy := minChunkY; cy <= maxChunkY; cy++ {
			chunk, exists := w.Chunks[[2]int{cx, cy}]
			if !exists {
				continue
			}

			for x := 0; x < chunkSize; x++ {
				for y := 0; y < chunkSize; y++ {
					worldX := cx*chunkSize + x
					worldY := cy*chunkSize + y

					if worldX < minX || worldX > maxX || worldY < minY || worldY > maxY {
						continue
					}

					block := chunk.Blocks[x][y]
					if block.Type != Air {
						img := ebiten.NewImage(blockSize, blockSize)
						img.Fill(BlockColors[block.Type])
						op := &ebiten.DrawImageOptions{}
						screenX := float64(worldX*blockSize) - w.CameraX
						screenY := float64(worldY*blockSize) - w.CameraY
						op.GeoM.Translate(screenX, screenY)
						screen.DrawImage(img, op)
					}
				}
			}
		}
	}
}
