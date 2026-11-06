// Gollatz Conjecture: three views of the Collatz conjecture.
//
//   - Path:        the numbers one starting value visits on its way to 1,
//     which you can also play as a melody.
//   - Histogram:   how many steps every number from 1 to N takes.
//   - Spectrogram: those step counts played as music (or raw static) and
//     run through an FFT.
package main

import (
	"bytes"
	"fmt"
	"image/color"
	"log"
	"math"
	"slices"
	"strconv"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font/gofont/goregular"
)

const (
	screenW = 1280
	screenH = 720

	plotX = 80
	plotY = 120
	plotW = screenW - plotX - 40
	plotH = screenH - plotY - 60

	maxPathStart   = 8_000_000_000 // 8,528,817,511 is the first path to overflow an int
	maxHistogramN  = 5_000_000
	maxSpectrogram = 500_000 // about a minute of static
	maxMusic       = 1_000   // about three minutes of music
)

type mode int

const (
	modePath mode = iota
	modeHistogram
	modeSpectrogram
)

var (
	modeNames   = []string{"Path", "Histogram", "Spectrogram"}
	modePrompts = []string{
		"Starting number:",
		"Count every number from 1 to:",
		"Play every number from 1 to:",
		"Play every number from 1 to:",
	}

	bg     = color.RGBA{18, 18, 24, 255}
	panel  = color.RGBA{36, 36, 48, 255}
	muted  = color.RGBA{150, 150, 165, 255}
	white  = color.RGBA{235, 235, 240, 255}
	red    = color.RGBA{230, 70, 70, 255}
	blue   = color.RGBA{90, 140, 240, 255}
	accent = color.RGBA{250, 180, 90, 255}

	face  *text.GoTextFace
	small *text.GoTextFace
)

// Input boxes. The Spectrogram view keeps separate numbers for static and
// music, since a good length for one is far too long for the other.
const (
	slotPath = iota
	slotHistogram
	slotStatic
	slotMusic
)

type Game struct {
	mode   mode
	music  bool // Spectrogram view: music (true) or raw static (false)
	inputs [4]string
	errMsg string

	// Path view
	sequence   []int
	pathPlayer *audio.Player
	pathDur    time.Duration

	// Histogram view
	histN      int
	counts     []int // counts[k] = how many numbers take k steps
	longestN   int
	longestLen int

	// Spectrogram view
	specN     int
	specMusic bool // whether the current picture and player are music or static
	specImg   *ebiten.Image
	specMaxHz int
	duration  time.Duration
	player    *audio.Player
	audioCtx  *audio.Context
}

func newGame() *Game {
	g := &Game{
		music:  true,
		inputs: [4]string{"27", "10000", "200000", "240"},
	}
	g.apply(modePath)
	g.apply(modeHistogram)
	return g
}

// ---------------------------------------------------------------- input

func (g *Game) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		g.switchTo((g.mode + 1) % 3)
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		for i := range modeNames {
			if x0, x1 := tabSpan(i); y < 44 && x >= x0 && x < x1 {
				g.switchTo(mode(i))
			}
		}
	}

	slot := g.slot(g.mode)
	for _, r := range ebiten.AppendInputChars(nil) {
		if r >= '0' && r <= '9' && len(g.inputs[slot]) < 13 {
			g.inputs[slot] += string(r)
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) && len(g.inputs[slot]) > 0 {
		g.inputs[slot] = g.inputs[slot][:len(g.inputs[slot])-1]
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeyNumpadEnter) {
		g.apply(g.mode)
	}

	switch g.mode {
	case modePath:
		if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
			if g.pathPlayer == nil && len(g.sequence) > 0 {
				g.buildPathMelody()
			}
			togglePlay(g.pathPlayer, g.pathDur)
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyR) && g.pathPlayer != nil {
			_ = g.pathPlayer.Rewind()
			g.pathPlayer.Play()
		}
	case modeSpectrogram:
		if inpututil.IsKeyJustPressed(ebiten.KeyM) {
			g.music = !g.music
			g.apply(modeSpectrogram)
		}
		if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
			togglePlay(g.player, g.duration)
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyR) && g.player != nil {
			_ = g.player.Rewind()
			g.player.Play()
		}
	}
	return nil
}

// slot says which input box a view is using right now.
func (g *Game) slot(m mode) int {
	switch {
	case m == modePath:
		return slotPath
	case m == modeHistogram:
		return slotHistogram
	case g.music:
		return slotMusic
	default:
		return slotStatic
	}
}

func togglePlay(p *audio.Player, dur time.Duration) {
	if p == nil {
		return
	}
	if p.IsPlaying() {
		p.Pause()
		return
	}
	if p.Position() >= dur {
		_ = p.Rewind()
	}
	p.Play()
}

// newPlayer swaps out an old player for a new one built from samples at rate.
// The sound card is only opened the first time something is played, so the
// app still works on a machine with no audio device.
func (g *Game) newPlayer(old *audio.Player, samples []float64, rate int) *audio.Player {
	if g.audioCtx == nil {
		g.audioCtx = audio.NewContext(playbackRate)
	}
	if old != nil {
		old.Pause()
		_ = old.Close()
	}
	return g.audioCtx.NewPlayerF32FromBytes(AudioBytes(samples, rate))
}

func (g *Game) buildPathMelody() {
	samples := Render(Melody(g.sequence, true), playbackRate)
	g.pathDur = time.Duration(len(samples)) * time.Second / playbackRate
	g.pathPlayer = g.newPlayer(g.pathPlayer, samples, playbackRate)
}

func (g *Game) switchTo(m mode) {
	g.mode = m
	g.errMsg = ""
	if m == modeSpectrogram && g.specImg == nil {
		g.apply(modeSpectrogram)
	}
	if m != modeSpectrogram && g.player != nil {
		g.player.Pause()
	}
	if m != modePath && g.pathPlayer != nil {
		g.pathPlayer.Pause()
	}
}

// apply reads the input box for a view and recalculates that view.
func (g *Game) apply(m mode) {
	g.errMsg = ""
	slot := g.slot(m)
	n, err := strconv.Atoi(g.inputs[slot])
	limits := []int{maxPathStart, maxHistogramN, maxSpectrogram, maxMusic}
	minimums := []int{1, 2, fftSize + 1, 8}
	switch {
	case err != nil || n < minimums[slot]:
		g.errMsg = fmt.Sprintf("Enter a whole number of at least %s.", commas(minimums[slot]))
		return
	case n > limits[slot]:
		g.errMsg = fmt.Sprintf("Keep it at or under %s.", commas(limits[slot]))
		return
	}

	switch m {
	case modePath:
		g.sequence = Sequence(n)
		if g.pathPlayer != nil {
			g.pathPlayer.Pause()
			_ = g.pathPlayer.Close()
			g.pathPlayer = nil // rebuilt the next time you press Space
		}

	case modeHistogram:
		steps := StoppingTimes(n)
		longest := slices.Max(steps)
		g.histN = n
		g.counts = make([]int, longest+1)
		g.longestLen = longest
		for i := 1; i <= n; i++ {
			g.counts[steps[i]]++
		}
		g.longestN = slices.Index(steps, longest) // first number with the most steps

	case modeSpectrogram:
		g.specN = n
		g.specMusic = g.music
		steps := StoppingTimes(n)
		if g.music {
			// One note per number. The spectrogram is worked out from a
			// low-quality copy of the music, and zoomed in to 0-2,000 Hz where
			// the notes are. You hear a full-quality copy.
			notes := Melody(steps[1:], false)
			sig := Render(notes, signalRate)
			g.specMaxHz = 2000
			g.specImg = ebiten.NewImageFromImage(SpectrogramImage(Spectrogram(sig), g.specMaxHz*fftSize/signalRate+1))
			audible := Render(notes, playbackRate)
			g.duration = time.Duration(len(audible)) * time.Second / playbackRate
			g.player = g.newPlayer(g.player, audible, playbackRate)
		} else {
			// The raw step counts, straight in as a sound wave.
			sig := Signal(steps)
			g.specMaxHz = signalRate / 2
			g.specImg = ebiten.NewImageFromImage(SpectrogramImage(Spectrogram(sig), fftSize/2+1))
			g.duration = time.Duration(len(sig)) * time.Second / signalRate
			g.player = g.newPlayer(g.player, sig, signalRate)
		}
	}
}

// ---------------------------------------------------------------- drawing

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(bg)
	g.drawTabs(screen)
	g.drawInput(screen)

	switch g.mode {
	case modePath:
		g.drawPath(screen)
	case modeHistogram:
		g.drawHistogram(screen)
	case modeSpectrogram:
		g.drawSpectrogram(screen)
	}
}

func tabSpan(i int) (int, int) {
	x := 20
	for j := 0; j < i; j++ {
		w, _ := text.Measure(modeNames[j], face, 0)
		x += int(w) + 40
	}
	w, _ := text.Measure(modeNames[i], face, 0)
	return x - 10, x + int(w) + 10
}

func (g *Game) drawTabs(screen *ebiten.Image) {
	for i, name := range modeNames {
		x0, x1 := tabSpan(i)
		if mode(i) == g.mode {
			vector.FillRect(screen, float32(x0), 8, float32(x1-x0), 30, panel, false)
			vector.FillRect(screen, float32(x0), 36, float32(x1-x0), 2, accent, false)
			drawText(screen, name, float64(x0+10), 14, white, face)
		} else {
			drawText(screen, name, float64(x0+10), 14, muted, face)
		}
	}
	drawText(screen, "Tab or click to switch views", screenW-230, 16, muted, small)
}

func (g *Game) drawInput(screen *ebiten.Image) {
	slot := g.slot(g.mode)
	prompt := modePrompts[slot]
	drawText(screen, prompt, 20, 62, white, face)
	pw, _ := text.Measure(prompt, face, 0)
	boxX := 30 + float32(pw)
	vector.FillRect(screen, boxX, 56, 160, 28, panel, false)
	cursor := ""
	if (time.Now().UnixMilli()/500)%2 == 0 {
		cursor = "|"
	}
	drawText(screen, g.inputs[slot]+cursor, float64(boxX)+8, 62, white, face)
	if g.errMsg != "" {
		drawText(screen, g.errMsg, float64(boxX)+172, 64, red, small)
	} else {
		drawText(screen, "press Enter", float64(boxX)+172, 64, muted, small)
	}
}

func (g *Game) drawPath(screen *ebiten.Image) {
	seq := g.sequence
	if len(seq) == 0 {
		return
	}
	peak := slices.Max(seq)
	drawText(screen, fmt.Sprintf("Starting at %s: %d steps to reach 1, peaking at %s.   Space: play it as a melody   R: restart",
		commas(seq[0]), len(seq)-1, commas(peak)), 20, 92, accent, face)

	// While the melody plays, the steps already heard turn orange.
	playing := -1
	if g.pathPlayer != nil && (g.pathPlayer.IsPlaying() || g.pathPlayer.Position() > 0) {
		playing = min(int(g.pathPlayer.Position().Seconds()*notesPerSecond), len(seq)-1)
	}
	drawAxes(screen)

	type pt struct{ x, y float32 }
	pts := make([]pt, len(seq))
	for i, v := range seq {
		x := float32(plotX)
		if len(seq) > 1 {
			x += float32(i) / float32(len(seq)-1) * plotW
		}
		y := float32(plotY+plotH) - float32(float64(v)/float64(peak))*plotH
		pts[i] = pt{x, y}
	}

	// Short paths get the original look: arrows between every step and a
	// label on each point. Long paths get a plain line so it stays readable.
	detailed := len(seq) <= 60
	for i := 0; i < len(pts)-1; i++ {
		a, b := pts[i], pts[i+1]
		lineClr := red
		if i < playing {
			lineClr = accent
		}
		vector.StrokeLine(screen, a.x, a.y, b.x, b.y, 1.5, lineClr, true)
		if detailed {
			angle := math.Atan2(float64(b.y-a.y), float64(b.x-a.x))
			for _, side := range []float64{-math.Pi / 6, math.Pi / 6} {
				vector.StrokeLine(screen, b.x, b.y,
					b.x-6*float32(math.Cos(angle+side)), b.y-6*float32(math.Sin(angle+side)),
					1.5, lineClr, true)
			}
		}
	}
	for i, p := range pts {
		if detailed || i == 0 || seq[i] == peak {
			vector.FillCircle(screen, p.x, p.y, 3, blue, true)
			drawText(screen, commas(seq[i]), float64(p.x)+5, float64(p.y)-16, white, small)
		}
	}
	if playing >= 0 {
		p := pts[playing]
		vector.FillCircle(screen, p.x, p.y, 7, accent, true)
		drawText(screen, commas(seq[playing]), float64(p.x)+10, float64(p.y)-26, white, face)
	}
	drawText(screen, "step →", plotX+plotW-50, plotY+plotH+10, muted, small)
}

func (g *Game) drawHistogram(screen *ebiten.Image) {
	if len(g.counts) == 0 {
		return
	}
	drawText(screen, fmt.Sprintf("1 to %s: the longest is %s, at %d steps",
		commas(g.histN), commas(g.longestN), g.longestLen), 20, 92, accent, face)
	drawAxes(screen)

	tallest := slices.Max(g.counts)
	barW := float32(plotW) / float32(len(g.counts))
	mx, my := ebiten.CursorPosition()
	hover := -1
	for k, c := range g.counts {
		h := float32(c) / float32(tallest) * plotH
		x := plotX + float32(k)*barW
		clr := blue
		if float32(mx) >= x && float32(mx) < x+barW && my >= plotY && my <= plotY+plotH {
			clr, hover = accent, k
		}
		vector.FillRect(screen, x, plotY+plotH-h, max(barW-1, 1), h, clr, false)
	}

	// A handful of step labels along the bottom.
	every := max(1, len(g.counts)/10)
	for k := 0; k < len(g.counts); k += every {
		drawText(screen, strconv.Itoa(k), float64(plotX+float32(k)*barW), plotY+plotH+8, muted, small)
	}
	drawText(screen, "steps to reach 1 →", plotX+plotW-140, plotY+plotH+26, muted, small)
	drawText(screen, commas(tallest), 10, plotY-4, muted, small)

	if hover >= 0 {
		label := fmt.Sprintf("%s numbers take %d steps", commas(g.counts[hover]), hover)
		w, _ := text.Measure(label, face, 0)
		x := min(float64(mx)+12, screenW-w-20)
		vector.FillRect(screen, float32(x)-6, float32(my)-30, float32(w)+12, 28, panel, false)
		drawText(screen, label, x, float64(my)-26, white, face)
	}
}

func (g *Game) drawSpectrogram(screen *ebiten.Image) {
	if g.specImg == nil {
		return
	}
	desc := fmt.Sprintf("1 to %s as music, one note per number (%s).   Space: play/pause   R: restart   M: hear it as raw static",
		commas(g.specN), g.duration.Round(time.Second))
	if !g.specMusic {
		desc = fmt.Sprintf("1 to %s as raw sound, %d per second (%s).   Space: play/pause   R: restart   M: hear it as music",
			commas(g.specN), signalRate, g.duration.Round(time.Second))
	}
	drawText(screen, desc, 20, 92, accent, face)

	b := g.specImg.Bounds()
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Scale(float64(plotW)/float64(b.Dx()), float64(plotH)/float64(b.Dy()))
	op.GeoM.Translate(plotX, plotY)
	screen.DrawImage(g.specImg, op)

	// Frequency labels on the left, seconds along the bottom.
	top := g.specMaxHz
	every := 1000
	if top <= 2000 {
		every = 250
	}
	for hz := 0; hz <= top; hz += every {
		y := float64(plotY+plotH) - float64(hz)/float64(top)*plotH
		drawText(screen, fmt.Sprintf("%d Hz", hz), 10, y-8, muted, small)
	}
	secs := g.duration.Seconds()
	stepSecs := math.Max(1, math.Round(secs/10))
	for s := 0.0; s <= secs; s += stepSecs {
		x := plotX + s/secs*plotW
		drawText(screen, fmt.Sprintf("%gs", s), x, plotY+plotH+8, muted, small)
	}

	// Playhead
	if g.player != nil && secs > 0 {
		pos := math.Min(g.player.Position().Seconds()/secs, 1)
		x := float32(plotX + pos*plotW)
		vector.StrokeLine(screen, x, plotY, x, plotY+plotH, 2, white, false)
	}
}

func drawAxes(screen *ebiten.Image) {
	vector.StrokeLine(screen, plotX, plotY+plotH, plotX+plotW, plotY+plotH, 1, muted, false)
	vector.StrokeLine(screen, plotX, plotY, plotX, plotY+plotH, 1, muted, false)
}

func drawText(dst *ebiten.Image, s string, x, y float64, clr color.Color, f *text.GoTextFace) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(clr)
	text.Draw(dst, s, f, op)
}

// commas formats 1234567 as "1,234,567".
func commas(n int) string {
	s := strconv.Itoa(n)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}

func (g *Game) Layout(_, _ int) (int, int) {
	return screenW, screenH
}

func main() {
	src, err := text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	if err != nil {
		log.Fatal(err)
	}
	face = &text.GoTextFace{Source: src, Size: 16}
	small = &text.GoTextFace{Source: src, Size: 13}

	ebiten.SetWindowSize(screenW, screenH)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowTitle("Gollatz Conjecture")
	if err := ebiten.RunGame(newGame()); err != nil {
		log.Fatal(err)
	}
}
