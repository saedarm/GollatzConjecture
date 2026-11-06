package main

import (
	"math"
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
	if got := len(AudioBytes(sig, signalRate)); got != want {
		t.Fatalf("got %d bytes, want %d", got, want)
	}
}

func TestMelodyStaysInScale(t *testing.T) {
	inScale := map[int]bool{}
	for i := range melodyRange {
		inScale[scaleNote(i)] = true
	}
	notes := Melody(Sequence(27), true)
	melody := 0
	for _, n := range notes {
		if n.Bass {
			continue
		}
		melody++
		midi := int(math.Round(69 + 12*math.Log2(n.Freq/440)))
		if !inScale[midi] {
			t.Fatalf("note %.1f Hz (MIDI %d) is outside the scale", n.Freq, midi)
		}
	}
	if melody != len(Sequence(27)) {
		t.Fatalf("got %d melody notes, want one per step (%d)", melody, len(Sequence(27)))
	}
}

func TestMelodyClimbsWithThePath(t *testing.T) {
	// 27's path peaks at 9,232, so that step should get the highest note
	// and the final 1 the lowest.
	seq := Sequence(27)
	notes := Melody(seq, true)[:len(seq)]
	peakAt := slices.Index(seq, 9232)
	for i, n := range notes {
		if n.Freq > notes[peakAt].Freq {
			t.Fatalf("step %d is higher than the peak", i)
		}
	}
	if notes[len(notes)-1].Freq != midiFreq(scaleNote(0)) {
		t.Fatalf("the final 1 should land on the lowest note")
	}
}

func TestRenderLength(t *testing.T) {
	notes := Melody([]int{1, 2, 3, 4, 5, 6, 7, 8}, false)
	samples := Render(notes, signalRate)
	wantSecs := 8/notesPerSecond + 1/notesPerSecond // last note rings for two beats
	if got := float64(len(samples)) / signalRate; math.Abs(got-wantSecs) > 0.01 {
		t.Fatalf("rendered %.3fs, want %.3fs", got, wantSecs)
	}
}
