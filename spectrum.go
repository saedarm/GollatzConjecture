package main

import (
	"encoding/binary"
	"image"
	"image/color"
	"math"

	"gonum.org/v1/gonum/dsp/fourier"
)

const (
	// signalRate is how many stopping times play per second when the
	// sequence is treated as sound. 8,000 is old-telephone quality.
	signalRate = 8000
	// playbackRate is the rate the audio device runs at. Each stopping time
	// is held for playbackRate/signalRate samples.
	playbackRate = 48000

	fftSize = 1024 // samples per FFT chunk
	hopSize = 256  // how far the chunk slides each time
	dbRange = 60.0 // how many decibels below the loudest point still get color
)

// Signal turns stopping times into a sound wave: centered on zero and scaled
// so the biggest swing reaches 1.
func Signal(steps []int) []float64 {
	vals := steps[1:]
	var mean float64
	for _, v := range vals {
		mean += float64(v)
	}
	mean /= float64(len(vals))

	sig := make([]float64, len(vals))
	var peak float64
	for i, v := range vals {
		sig[i] = float64(v) - mean
		peak = math.Max(peak, math.Abs(sig[i]))
	}
	if peak > 0 {
		for i := range sig {
			sig[i] /= peak
		}
	}
	return sig
}

// Spectrogram slides an FFT along the signal and returns one row per chunk.
// Each row holds the loudness in decibels of every frequency bucket, from
// 0 Hz up to half of signalRate.
func Spectrogram(sig []float64) [][]float64 {
	fft := fourier.NewFFT(fftSize)

	// A Hann window fades each chunk in and out so that cutting a wave off
	// mid-cycle doesn't smear energy across every bucket.
	window := make([]float64, fftSize)
	for i := range window {
		window[i] = 0.5 - 0.5*math.Cos(2*math.Pi*float64(i)/float64(fftSize-1))
	}

	chunk := make([]float64, fftSize)
	coeffs := make([]complex128, fftSize/2+1)
	var rows [][]float64
	for start := 0; start+fftSize <= len(sig); start += hopSize {
		for i := range chunk {
			chunk[i] = sig[start+i] * window[i]
		}
		coeffs = fft.Coefficients(coeffs, chunk)

		row := make([]float64, len(coeffs))
		for k, c := range coeffs {
			power := real(c)*real(c) + imag(c)*imag(c)
			row[k] = 10 * math.Log10(power+1e-12)
		}
		rows = append(rows, row)
	}
	return rows
}

// SpectrogramImage draws the spectrogram as a picture: time runs left to
// right, frequency runs bottom to top, and brighter means louder. Only the
// first `buckets` frequency buckets are drawn, so a picture of music can zoom
// in on the range where the notes are.
func SpectrogramImage(rows [][]float64, buckets int) *image.RGBA {
	if len(rows) == 0 {
		return image.NewRGBA(image.Rect(0, 0, 1, 1))
	}
	w, h := len(rows), min(buckets, len(rows[0]))
	for i := range rows {
		rows[i] = rows[i][:h]
	}

	loudest := math.Inf(-1)
	for _, row := range rows {
		for _, v := range row {
			loudest = math.Max(loudest, v)
		}
	}

	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for x, row := range rows {
		for k, v := range row {
			t := (v - (loudest - dbRange)) / dbRange
			img.Set(x, h-1-k, heat(t))
		}
	}
	return img
}

// heat maps 0..1 onto a black → purple → orange → pale yellow scale.
func heat(t float64) color.RGBA {
	stops := []color.RGBA{
		{0, 0, 4, 255},
		{80, 18, 123, 255},
		{182, 54, 121, 255},
		{251, 136, 97, 255},
		{252, 253, 191, 255},
	}
	t = math.Max(0, math.Min(1, t))
	pos := t * float64(len(stops)-1)
	i := int(pos)
	if i >= len(stops)-1 {
		return stops[len(stops)-1]
	}
	f := pos - float64(i)
	a, b := stops[i], stops[i+1]
	mix := func(x, y uint8) uint8 { return uint8(float64(x) + f*(float64(y)-float64(x))) }
	return color.RGBA{mix(a.R, b.R), mix(a.G, b.G), mix(a.B, b.B), 255}
}

// AudioBytes packs a signal recorded at rate (which must divide playbackRate)
// in the format Ebiten's audio player expects: 32-bit float, little endian,
// two channels, at playbackRate.
func AudioBytes(sig []float64, rate int) []byte {
	repeat := playbackRate / rate
	const volume = 0.4
	out := make([]byte, 0, len(sig)*repeat*2*4)
	var buf [4]byte
	for _, s := range sig {
		binary.LittleEndian.PutUint32(buf[:], math.Float32bits(float32(s*volume)))
		for range repeat {
			out = append(out, buf[:]...) // left
			out = append(out, buf[:]...) // right
		}
	}
	return out
}
