package main

import (
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/effects"
	"github.com/faiface/beep/speaker"
)

type Sound struct {
	Stream  beep.StreamSeeker
	Format  beep.Format
	volume  *effects.Volume
	control *beep.Ctrl
	Empty   bool
}

func NewSound(stream beep.StreamSeeker, format beep.Format) *Sound {

	sound := &Sound{
		Stream: stream,
		Format: format,
	}

	sound.ReloadStream()

	speaker.Play(sound.volume)

	return sound

}

func (sound *Sound) ReloadStream() {

	// if globals.Settings.Get(SettingsCacheAudioBeforePlayback).AsBool() {

	// 	ogStream := sound.Stream

	// 	// globals.EventLog.Log("caching audio", false)

	// 	buffer := beep.NewBuffer(sound.Format)
	// 	buffer.Append(ogStream)
	// 	sound.Stream = buffer.Streamer(0, ogStream.Len())

	// 	// Close the stream if possible before replacing it, we don't need it after buffering
	// 	if stream, ok := ogStream.(beep.StreamSeekCloser); ok {
	// 		stream.Close()
	// 	}

	// }

	resampled := beep.Resample(3, sound.Format.SampleRate, globals.ChosenAudioSampleRate, sound.Stream)

	seq := beep.Seq(resampled, beep.Callback(func() {
		sound.Empty = true
	}))

	sound.control = &beep.Ctrl{
		Streamer: seq,
		Paused:   true,
	}

	sound.volume = &effects.Volume{
		Streamer: sound.control,
		Volume:   0,
		Base:     100,
		Silent:   false,
	}

	sound.UpdateVolume()

}

func (sound *Sound) UpdateVolume() {
	speaker.Lock()

	v := globals.Settings.Get(SettingsAudioVolume).AsFloat()
	if v > 0 {
		sound.volume.Silent = false
		sound.volume.Volume = (v / 100) - 1
	} else {
		sound.volume.Silent = true
	}

	speaker.Unlock()
}

func (sound *Sound) Play() {
	if sound.Stream == nil {
		sound.ReloadStream()
	}
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

func (sound *Sound) Seek(to time.Duration) {
	for to > sound.Length() {
		to = to - sound.Length()
	}
	speaker.Lock()
	sound.Stream.Seek(sound.Format.SampleRate.N(to))
	speaker.Unlock()
}

func (sound *Sound) SeekPercentage(percentage float32) {
	speaker.Lock()
	sound.Stream.Seek(int(percentage * float32(sound.Stream.Len())))
	speaker.Unlock()
}

func (sound *Sound) Length() time.Duration {
	speaker.Lock()
	d := sound.Format.SampleRate.D(sound.Stream.Len())
	speaker.Unlock()
	return d
}

func (sound *Sound) Position() time.Duration {
	speaker.Lock()
	d := sound.Format.SampleRate.D(sound.Stream.Position())
	speaker.Unlock()
	return d
}

func (sound *Sound) Destroy() {
	sound.Pause()
	speaker.Lock()

	if stream, ok := sound.Stream.(beep.StreamSeekCloser); ok {
		stream.Close()
	}
	sound.Stream = nil

	speaker.Unlock()
}

// func (sound *Sound) TogglePause() {
// 	if sound.Control.Paused {
// 		sound.Play()
// 	} else {
// 		sound.Pause()
// 	}
// }
