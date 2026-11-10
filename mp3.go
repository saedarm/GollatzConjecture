package main

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"path/filepath"

	"github.com/braheezy/shine-mp3/pkg/mp3"
)

// mp3Dir is where every song the app makes gets saved. It's relative to
// wherever the app was started: next to gollatz.exe if you double-click it,
// or the project folder if you use `go run .`.
const mp3Dir = "mp3"

// autoSave writes the song to mp3/<name>.mp3 in the background and reports
// back on done. The same name always means the same song, so a file that
// already exists is left alone rather than encoded again.
func autoSave(name string, samples []float64, rate int, done chan<- string) {
	path := filepath.Join(mp3Dir, name+".mp3")
	if _, err := os.Stat(path); err == nil {
		done <- "Already saved: " + absPath(path)
		return
	}
	go func() {
		if err := SaveMP3(path, samples, rate); err != nil {
			done <- "Couldn't save the MP3: " + err.Error()
			return
		}
		done <- "Saved " + absPath(path)
	}()
}

// SaveMP3 encodes samples recorded at rate (which must divide playbackRate)
// as an MP3 at playbackRate. It's saved in stereo, with the same sound on
// both sides, because the encoder's mono mode writes broken frames.
func SaveMP3(path string, samples []float64, rate int) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	// The encoder works in whole frames of 1,152 samples, so the song is
	// padded with silence: up to the next whole frame, plus two more so the
	// last note isn't clipped.
	const frame = 1152
	repeat := playbackRate / rate
	n := len(samples) * repeat
	padded := (n/frame + 3) * frame
	pcm := make([]int16, padded*2) // left, right, left, right...
	for i, s := range samples {
		v := int16(math.Max(-1, math.Min(1, s)) * 32000)
		for j := range repeat {
			k := (i*repeat + j) * 2
			pcm[k], pcm[k+1] = v, v
		}
	}

	// Write to a temporary name first so a half-written file never looks
	// like a finished song.
	tmp := path + ".part"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	w := bufio.NewWriter(f)
	if err := mp3.NewEncoder(playbackRate, 2).Write(w, pcm); err != nil {
		f.Close()
		os.Remove(tmp)
		return fmt.Errorf("encoding: %w", err)
	}
	if err := w.Flush(); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, path)
}

func absPath(p string) string {
	if a, err := filepath.Abs(p); err == nil {
		return a
	}
	return p
}
