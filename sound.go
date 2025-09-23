package main

import (
	"io/fs"
	"math/rand/v2"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/effects"
	"github.com/faiface/beep/speaker"
)

const (
	AudioChannelSoundCard = iota
	AudioChannelUI
)

type Sound struct {
	Stream        beep.StreamSeeker
	Format        beep.Format
	Empty         bool
	filepath      string
	limitPlayback bool
	volume        *effects.Volume
	control       *beep.Ctrl
	channel       int
}

func NewSound(stream beep.StreamSeeker, format beep.Format, channel int) *Sound {

	sound := &Sound{
		Stream:  stream,
		Format:  format,
		channel: channel,
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

	var v float64
	switch sound.channel {
	case AudioChannelSoundCard:
		v = globals.Settings.Get(SettingsAudioSoundVolume).AsFloat()
	case AudioChannelUI:
		v = globals.Settings.Get(SettingsAudioUIVolume).AsFloat()
	}

	if v > 0 {
		sound.volume.Silent = false
		sound.volume.Volume = v - 1
	} else {
		sound.volume.Silent = true
	}

	speaker.Unlock()
}

func (sound *Sound) Play() {

	if sound.limitPlayback {
		playingSounds[sound.filepath] = time.Now()
	}

	if sound.Stream == nil {
		sound.ReloadStream()
	}
	speaker.Lock()
	sound.control.Paused = false
	speaker.Unlock()
}

func (s *Sound) IsEmpty() bool {
	return s.Stream.Position() == s.Stream.Len()
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

type UISoundType string

var uiSounds = map[UISoundType][]string{}

func init() {

	filepath.Walk("assets/sounds/snddev_sine/", func(path string, info fs.FileInfo, err error) error {

		if info.IsDir() {
			return nil
		}

		filename := filepath.Base(path)
		ext := filepath.Ext(path)
		fnNoExt := ""

		ni := strings.LastIndex(filename, "_")

		numberedPortion := ""

		if ni >= 0 {
			numberedPortion = filename[ni+1 : strings.Index(filename, ext)]
			fnNoExt = filename[:ni]
		}

		numbered := false

		// Not "sound_type_00.wav", but rather "sound_type.wav"; not numbered
		if _, err := strconv.Atoi(numberedPortion); err == nil {
			numbered = true
		}

		if !numbered {
			fnNoExt = filename[:strings.Index(filename, ext)]
		}

		if _, ok := uiSounds[UISoundType(fnNoExt)]; !ok {
			uiSounds[UISoundType(fnNoExt)] = []string{}
		}

		uiSounds[UISoundType(fnNoExt)] = append(uiSounds[UISoundType(fnNoExt)], path)

		return nil
	})

}

const (
	UISoundTypeSelect    UISoundType = "select"
	UISoundTypeTap                   = "tap"
	UISoundTypeSwipe                 = "swipe"
	UISoundTypeType                  = "type"
	UISoundTypeToggleOff             = "toggle_off"
	UISoundTypeToggleOn              = "toggle_on"
	UISoundTypeProgress              = "progress_loop"
)

func PlayUISound(soundType UISoundType) {

	snd, _ := globals.Resources.Get(uiSounds[soundType][rand.IntN(len(uiSounds[soundType]))]).AsNewSound(true, AudioChannelUI)
	if snd != nil {
		snd.Play()
	}

}
