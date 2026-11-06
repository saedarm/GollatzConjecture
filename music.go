package main

import "math"

const (
	notesPerSecond = 6.0 // melody tempo
	notesPerBar    = 8   // the bass changes chord every 8 melody notes
	melodyRange    = 15  // three octaves of the scale below
)

// The major pentatonic scale (C D E G A). It has no notes that clash with
// each other, so any order of them sounds like a tune rather than a mistake.
var pentatonic = []int{0, 2, 4, 7, 9}

// One bass chord per bar, cycling C, A minor, F, G: the four chords behind
// a huge number of pop songs. Values are MIDI note numbers (60 = middle C).
var progression = []int{48, 45, 41, 43}

// Note is one sound: a pitch that starts at a time and lasts a while.
type Note struct {
	Freq, Start, Dur, Vol float64
	Bass                  bool
}

func midiFreq(m int) float64 { return 440 * math.Pow(2, float64(m-69)/12) }

// scaleNote returns the MIDI note for step i of the scale, where 0 is middle C.
func scaleNote(i int) int {
	return 60 + 12*(i/len(pentatonic)) + pentatonic[i%len(pentatonic)]
}

// Melody turns a list of numbers into notes, one per number. Bigger numbers
// play higher. With logScale, pitch follows the number of digits rather than
// the raw size, which suits Collatz paths that shoot from 27 up to 9,232.
// A bass line plays underneath so it sounds like a song, not a ringtone.
func Melody(values []int, logScale bool) []Note {
	if len(values) == 0 {
		return nil
	}
	size := func(v int) float64 {
		if logScale {
			return math.Log2(float64(max(v, 1)))
		}
		return float64(v)
	}
	lo, hi := math.Inf(1), math.Inf(-1)
	for _, v := range values {
		lo, hi = math.Min(lo, size(v)), math.Max(hi, size(v))
	}

	beat := 1 / notesPerSecond
	notes := make([]Note, 0, len(values)+len(values)/notesPerBar*2+2)
	for i, v := range values {
		t := 0.0
		if hi > lo {
			t = (size(v) - lo) / (hi - lo)
		}
		step := int(math.Round(t * (melodyRange - 1)))
		notes = append(notes, Note{
			Freq: midiFreq(scaleNote(step)), Start: float64(i) * beat, Dur: beat * 2, Vol: 0.5,
		})
	}

	bars := (len(values) + notesPerBar - 1) / notesPerBar
	for b := range bars {
		root := progression[b%len(progression)]
		start, dur := float64(b*notesPerBar)*beat, notesPerBar*beat
		notes = append(notes,
			Note{Freq: midiFreq(root), Start: start, Dur: dur, Vol: 0.35, Bass: true},
			Note{Freq: midiFreq(root + 7), Start: start, Dur: dur, Vol: 0.15, Bass: true},
		)
	}
	return notes
}

// Render plays the notes into a list of samples at the given sample rate,
// scaled so the loudest moment reaches 0.9.
func Render(notes []Note, rate int) []float64 {
	var end float64
	for _, n := range notes {
		end = math.Max(end, n.Start+n.Dur)
	}
	out := make([]float64, int(end*float64(rate))+1)
	r := float64(rate)

	for _, n := range notes {
		first := int(n.Start * r)
		count := int(n.Dur * r)
		step := 2 * math.Pi * n.Freq / r
		for j := 0; j < count && first+j < len(out); j++ {
			t := float64(j) / r
			left := n.Dur - t
			var env, wave float64
			phase := step * float64(j)
			if n.Bass {
				// Soft swell in and out, like a held organ note.
				env = math.Min(t/0.03, 1) * math.Min(left/0.05, 1)
				wave = math.Sin(phase) + 0.2*math.Sin(2*phase)
			} else {
				// Quick strike then a fade, like a marimba or a plucked string.
				env = math.Min(t/0.005, 1) * math.Exp(-t*6) * math.Min(left/0.02, 1)
				wave = math.Sin(phase) + 0.4*math.Sin(2*phase) + 0.15*math.Sin(3*phase)
			}
			out[first+j] += n.Vol * env * wave
		}
	}

	var peak float64
	for _, v := range out {
		peak = math.Max(peak, math.Abs(v))
	}
	if peak > 0 {
		for i := range out {
			out[i] *= 0.9 / peak
		}
	}
	return out
}
