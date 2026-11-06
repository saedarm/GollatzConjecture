package main

import (
	"slices"
	"testing"
)

func TestSequence27(t *testing.T) {
	seq := Sequence(27)
	if steps := len(seq) - 1; steps != 111 {
		t.Fatalf("27 took %d steps, want 111", steps)
	}
	if peak := slices.Max(seq); peak != 9232 {
		t.Fatalf("27 peaked at %d, want 9232", peak)
	}
}

func TestStoppingTimesMatchSequence(t *testing.T) {
	steps := StoppingTimes(5000)
	for n := 1; n <= 5000; n++ {
		if want := len(Sequence(n)) - 1; steps[n] != want {
			t.Fatalf("StoppingTimes[%d] = %d, want %d", n, steps[n], want)
		}
	}
}

func TestSpectrogramShape(t *testing.T) {
	sig := Signal(StoppingTimes(20000))
	rows := Spectrogram(sig)
	wantRows := (len(sig)-fftSize)/hopSize + 1
	if len(rows) != wantRows {
		t.Fatalf("got %d rows, want %d", len(rows), wantRows)
	}
	if len(rows[0]) != fftSize/2+1 {
		t.Fatalf("got %d buckets per row, want %d", len(rows[0]), fftSize/2+1)
	}
}

func TestAudioBytesLength(t *testing.T) {
	sig := []float64{0, 0.5, -0.5}
	want := len(sig) * (playbackRate / signalRate) * 2 * 4
	if got := len(AudioBytes(sig)); got != want {
		t.Fatalf("got %d bytes, want %d", got, want)
	}
}
