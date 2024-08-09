package main

import (
    "fmt"
    "image/color"
    "log"
    "strconv"
    "math"

    "github.com/hajimehoshi/ebiten/v2"
    "github.com/hajimehoshi/ebiten/v2/vector"
    "github.com/hajimehoshi/ebiten/v2/inpututil"
    "golang.org/x/image/font"
    "golang.org/x/image/font/opentype"
    "golang.org/x/image/font/gofont/goregular"
    "github.com/hajimehoshi/ebiten/v2/text"
    
)

const (
    minWidth      = 800
    minHeight     = 600
    inputBoxWidth = 200
    inputBoxHeight = 30
    buttonWidth   = 100
    buttonHeight  = 30
    padding       = 50
)

type Point struct {
    X, Y float32
}

type Game struct {
    font       font.Face
    input      string
    calculated bool
    sequence   []int
    points     []Point
    width      int
    height     int
}

func (g *Game) Update() error {
    if !g.calculated {
        g.handleInput()
    }
    return nil
}

func (g *Game) handleInput() {
    runes := ebiten.AppendInputChars(nil)
    g.input += string(runes)

    if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) && len(g.input) > 0 {
        g.input = g.input[:len(g.input)-1]
    }

    if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || (inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) && g.isMouseOverButton()) {
        g.calculateCollatz()
    }
}

func (g *Game) isMouseOverButton() bool {
    x, y := ebiten.CursorPosition()
    return x >= g.width/2-buttonWidth/2 && x <= g.width/2+buttonWidth/2 &&
           y >= 100 && y <= 100+buttonHeight
}

func (g *Game) calculateCollatz() {
    input, err := strconv.Atoi(g.input)
    if err != nil || input < 1 {
        g.input = "Invalid input"
        return
    }
    
    g.sequence = CollatzConjecture(input)
    g.calculated = true
    g.adjustWindowSize()
    g.calculatePoints()
}

func (g *Game) adjustWindowSize() {
    g.width = max(minWidth, len(g.sequence)*20 + 2*padding)
    g.height = minHeight
    ebiten.SetWindowSize(g.width, g.height)
}

func (g *Game) calculatePoints() {
    g.points = make([]Point, len(g.sequence))
    maxVal := float32(g.sequence[0])
    for _, v := range g.sequence {
        if float32(v) > maxVal {
            maxVal = float32(v)
        }
    }

    for i, v := range g.sequence {
        x := float32(i) / float32(len(g.sequence)-1) * float32(g.width - 2*padding) + float32(padding)
        y := (1 - float32(v)/maxVal) * float32(g.height - 2*padding) + float32(padding)
        g.points[i] = Point{X: x, Y: y}
    }
}

func (g *Game) Draw(screen *ebiten.Image) {
    if !g.calculated {
        g.drawInputUI(screen)
    } else {
        g.drawGraph(screen)
    }
}

func (g *Game) drawInputUI(screen *ebiten.Image) {
    vector.DrawFilledRect(screen, float32(g.width/2-inputBoxWidth/2), 50, float32(inputBoxWidth), float32(inputBoxHeight), color.White, false)
    text.Draw(screen, g.input, g.font, g.width/2-inputBoxWidth/2+5, 75, color.Black)

    vector.DrawFilledRect(screen, float32(g.width/2-buttonWidth/2), 100, float32(buttonWidth), float32(buttonHeight), color.RGBA{100, 200, 100, 255}, false)
    text.Draw(screen, "Enter", g.font, g.width/2-30, 125, color.Black)

    text.Draw(screen, "Enter a positive integer:", g.font, g.width/2-100, 30, color.White)
}

func (g *Game) drawGraph(screen *ebiten.Image) {
    for i := 0; i < len(g.points)-1; i++ {
        vector.StrokeLine(screen, g.points[i].X, g.points[i].Y, g.points[i+1].X, g.points[i+1].Y, 1, color.RGBA{255, 0, 0, 255}, false)
        
        // Draw arrowhead
        angle := math.Atan2(float64(g.points[i+1].Y-g.points[i].Y), float64(g.points[i+1].X-g.points[i].X))
        arrowSize := float32(5)
        vector.StrokeLine(screen, 
            g.points[i+1].X, g.points[i+1].Y,
            g.points[i+1].X - arrowSize*float32(math.Cos(angle-math.Pi/6)), g.points[i+1].Y - arrowSize*float32(math.Sin(angle-math.Pi/6)),
            1, color.RGBA{255, 0, 0, 255}, false)
        vector.StrokeLine(screen, 
            g.points[i+1].X, g.points[i+1].Y,
            g.points[i+1].X - arrowSize*float32(math.Cos(angle+math.Pi/6)), g.points[i+1].Y - arrowSize*float32(math.Sin(angle+math.Pi/6)),
            1, color.RGBA{255, 0, 0, 255}, false)
    }

    for i, point := range g.points {
        vector.DrawFilledCircle(screen, point.X, point.Y, 3, color.RGBA{0, 0, 255, 255}, false)
        text.Draw(screen, fmt.Sprintf("%d", g.sequence[i]), g.font, int(point.X)+5, int(point.Y), color.White)
    }

    text.Draw(screen, fmt.Sprintf("Steps: %d", len(g.sequence)-1), g.font, 10, 20, color.White)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
    return g.width, g.height
}

func CollatzConjecture(n int) []int {
    sequence := []int{n}
    for n != 1 {
        if n%2 == 0 {
            n = n / 2
        } else {
            n = 3*n + 1
        }
        sequence = append(sequence, n)
    }
    return sequence
}

func main() {
    tt, err := opentype.Parse(goregular.TTF)
    if err != nil {
        log.Fatal(err)
    }

    font, err := opentype.NewFace(tt, &opentype.FaceOptions{
        Size:    12,
        DPI:     72,
        Hinting: font.HintingFull,
    })
    if err != nil {
        log.Fatal(err)
    }

    game := &Game{
        font:  font,
        width: minWidth,
        height: minHeight,
    }

    ebiten.SetWindowSize(game.width, game.height)
    ebiten.SetWindowTitle("Collatz Conjecture Graph")

    if err := ebiten.RunGame(game); err != nil {
        log.Fatal(err)
    }
}

func max(a, b int) int {
    if a > b {
        return a
    }
    return b
}