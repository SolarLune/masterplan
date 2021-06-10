package main

import (
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/effects"
	"github.com/faiface/beep/speaker"
)

type Sound struct {
	Stream           beep.StreamSeekCloser
	Format           beep.Format
	volume           effects.Volume
	control          *beep.Ctrl
	PlaybackFinished func()
}

func NewSound(stream beep.StreamSeekCloser, format beep.Format) *Sound {

	sound := &Sound{
		Stream: stream,
		Format: format,
	}

	loop := beep.Loop(-1, stream)
	resampled := beep.Resample(3, format.SampleRate, 44100, loop)
	seq := beep.Seq(resampled, beep.Callback(sound.PlaybackFinished))
	volume := effects.Volume{
		Streamer: seq,
		Volume:   0,
		Base:     2,
		Silent:   false,
	}

	sound.control = &beep.Ctrl{
		Streamer: volume.Streamer,
		Paused:   true,
	}

	speaker.Play(sound.control)

	return sound

}

func (sound *Sound) Play() {
	speaker.Lock()
	sound.control.Paused = false
	speaker.Unlock()
}

func (sound *Sound) Pause() {
	speaker.Lock()
	sound.control.Paused = true
	speaker.Unlock()
}

func (sound *Sound) IsPaused() bool {
	return sound.control.Paused
}

func (sound *Sound) SeekPercentage(percentage float32) {
	sound.Stream.Seek(int(percentage * float32(sound.Stream.Len())))
}

func (sound *Sound) Length() time.Duration {
	return sound.Format.SampleRate.D(sound.Stream.Len())
}

func (sound *Sound) Position() time.Duration {
	return sound.Format.SampleRate.D(sound.Stream.Position())
}

// func (sound *Sound) TogglePause() {
// 	if sound.Control.Paused {
// 		sound.Play()
// 	} else {
// 		sound.Pause()
// 	}
// }
