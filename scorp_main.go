package main

import (
	"fmt"
	"image"
	_ "image/png"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"image/color"
)

const (
	screenWidth     = 640
	screenHeight    = 480
	gridSize        = 32
	gridWidth       = screenWidth / gridSize
	gridHeight      = screenHeight / gridSize
	playerSpeed     = 5
	numLanes        = 5
	numCarsPerLane  = 6 // Reduced number for clearer spacing
	carSpeedMin     = 1.5
	carSpeedMax     = 3.0
	minCarGap       = 3   // Minimum gap between cars in grid units
	maxCarGap       = 6   // Maximum gap between cars in grid units
)

type GameObject struct {
	x, y    float64
	speed   float64
	image   *ebiten.Image
	width   int
	height  int
	isRight bool
}

type Game struct {
	player         *GameObject
	background     *ebiten.Image
	objects        map[string]*ebiten.Image
	cars           []*GameObject
	currentTime    int
	lastUpdateTime time.Time
	gameState      string
}

func NewGame() *Game {
	g := &Game{
		currentTime:    60,
		lastUpdateTime: time.Now(),
		objects:        make(map[string]*ebiten.Image),
		gameState:      "playing",
		player:         &GameObject{}, // Initialize the player
	}
	g.loadImages()
	g.initializeGame()
	return g
}

func (g *Game) loadImages() {
	var err error

	// Load player image
	g.player.image, _, err = ebitenutil.NewImageFromFile("player.png")
	if err != nil {
		log.Printf("Error loading player.png: %v", err)
		fileInfo, err := os.Stat("player.png")
		if err != nil {
			log.Printf("Error getting file info for player.png: %v", err)
		} else {
			log.Printf("File info for player.png: Size: %d bytes, Permissions: %v", fileInfo.Size(), fileInfo.Mode())
		}
	} else {
		bounds := g.player.image.Bounds()
		log.Printf("Successfully loaded player.png. Dimensions: %dx%d", bounds.Dx(), bounds.Dy())
		g.player.width = bounds.Dx()
		g.player.height = bounds.Dy()
	}

	// Load background image
	g.background, _, err = ebitenutil.NewImageFromFile("background.png")
	if err != nil {
		log.Printf("Error loading background.png: %v", err)
	} else {
		log.Println("Successfully loaded background.png")
	}

	// Load object images
	objectTypes := []string{"car"}
	for _, objType := range objectTypes {
		filename := objType + ".png"
		g.objects[objType], _, err = ebitenutil.NewImageFromFile(filename)
		if err != nil {
			log.Printf("Error loading %s: %v", filename, err)
		} else {
			log.Printf("Successfully loaded %s", filename)
		}
	}

	// Check if all required images are loaded
	if g.player.image == nil {
		log.Println("Warning: Player image not loaded. Using placeholder.")
		g.player.width = gridSize
		g.player.height = gridSize
	}
	if g.background == nil {
		log.Println("Warning: Background image not loaded. Game may not display correctly.")
	}
	if g.objects["car"] == nil {
		log.Println("Warning: Some object images not loaded. Game may not display correctly.")
	}
}

func (g *Game) initializeGame() {
	// Set initial player position
	g.player.x = float64(gridWidth / 2 * gridSize)
	g.player.y = float64((gridHeight - 1) * gridSize)

	// Clear existing cars
	g.cars = []*GameObject{}

	// Initialize more cars in each lane
	for lane := 0; lane < numLanes; lane++ {
		lastCarX := -float64(gridSize) // Start position before the screen
		for i := 0; i < numCarsPerLane; i++ {
			// Calculate gap between cars
			minGap := lastCarX + float64(minCarGap*gridSize)
			maxGap := lastCarX + float64(maxCarGap*gridSize)
			carX := minGap + rand.Float64()*(maxGap-minGap)

			g.cars = append(g.cars, &GameObject{
				x:       carX,
				y:       float64(5+lane) * gridSize,  // Different lanes for cars
				speed:   carSpeedMin + rand.Float64()*(carSpeedMax-carSpeedMin),
				image:   g.objects["car"],
				width:   gridSize * 2,
				height:  gridSize,
				isRight: rand.Intn(2) == 0,
			})

			lastCarX = carX
		}
	}
}

func (g *Game) Update() error {
	if g.gameState != "playing" {
		if ebiten.IsKeyPressed(ebiten.KeySpace) {
			g.initializeGame()
			g.currentTime = 60
			g.gameState = "playing"
		}
		return nil
	}

	now := time.Now()
	elapsed := now.Sub(g.lastUpdateTime).Seconds()
	g.lastUpdateTime = now

	if ebiten.IsKeyPressed(ebiten.KeyArrowLeft) {
		g.player.x -= gridSize * elapsed * playerSpeed
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowRight) {
		g.player.x += gridSize * elapsed * playerSpeed
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowUp) {
		g.player.y -= gridSize * elapsed * playerSpeed
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowDown) {
		g.player.y += gridSize * elapsed * playerSpeed
	}

	g.player.x = clamp(g.player.x, 0, screenWidth-gridSize)
	g.player.y = clamp(g.player.y, 0, screenHeight-gridSize)

	// Update cars
	for _, car := range g.cars {
		if car.isRight {
			car.x += car.speed * elapsed * gridSize
			if car.x > screenWidth {
				car.x = -float64(car.width)
			}
		} else {
			car.x -= car.speed * elapsed * gridSize
			if car.x < -float64(car.width) {
				car.x = screenWidth
			}
		}
	}

	// Check collisions
	g.checkCollisions()

	// Update time
	g.currentTime -= int(elapsed)
	if g.currentTime <= 0 {
		g.gameState = "lose"
	}

	// Check win condition
	if g.player.y <= 0 {
		g.gameState = "win"
	}

	return nil
}

func (g *Game) checkCollisions() {
	playerRect := image.Rect(int(g.player.x), int(g.player.y), int(g.player.x)+g.player.width, int(g.player.y)+g.player.height)

	// Check car collisions
	for _, car := range g.cars {
		carRect := image.Rect(int(car.x), int(car.y), int(car.x)+car.width, int(car.y)+car.height)
		if playerRect.Overlaps(carRect) {
			g.gameState = "lose"
			return
		}
	}
}

func (g *Game) Draw(screen *ebiten.Image) {
	// Draw background
	op := &ebiten.DrawImageOptions{}
	screen.DrawImage(g.background, op)

	// Draw cars
	for _, car := range g.cars {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(car.x, car.y)
		screen.DrawImage(car.image, op)
	}

	// Draw player
	if g.player.image != nil {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(g.player.x, g.player.y)
		screen.DrawImage(g.player.image, op)
	} else {
		log.Println("Player image is nil, cannot draw")
		// Draw a placeholder rectangle for the player
		vector.DrawFilledRect(screen,
			float32(g.player.x),
			float32(g.player.y),
			float32(g.player.width),
			float32(g.player.height),
			color.RGBA{255, 0, 0, 255},
			false)
	}

	// Draw time
	ebitenutil.DebugPrint(screen, fmt.Sprintf("Time: %d", g.currentTime))

	// Draw game state
	if g.gameState == "win" {
		ebitenutil.DebugPrint(screen, "\n\nYou Win! Press SPACE to restart")
	} else if g.gameState == "lose" {
		ebitenutil.DebugPrint(screen, "\n\nGame Over! Press SPACE to restart")
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Scorpions and Ferraris")

	if err := ebiten.RunGame(NewGame()); err != nil {
		log.Fatal(err)
	}
}

func clamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
