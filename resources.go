package main

import (
	"image/gif"
	"log"
	"os"
	"strings"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/flac"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/vorbis"
	"github.com/faiface/beep/wav"
	"github.com/gabriel-vasile/mimetype"
	rl "github.com/gen2brain/raylib-go/raylib"
)

type Resource struct {
	ModTime time.Time
	// Path facing the object requesting the resouce (e.g. "~/home/pictures/test.png" or "https://solarlune.com/media/bartender.png")
	ResourcePath string

	// Where the resource is located on disk (e.g. "~/home/pictures/test.png" or "/tmp/masterplan_resource076755900.png", for downloaded resources)
	LocalFilepath string

	// Pointer to the data the resource stands for (e.g. a rl.Texture2D for an image)
	Data interface{}

	// Whether or not the resource is located in the temporary directory
	// (e.g. was downloaded by MasterPlan, and so should be deleted after usage).
	Temporary bool

	// MIME data for the Resource.
	MimeData *mimetype.MIME
}

func (project *Project) RegisterResource(resourcePath, localFilepath string, data interface{}) *Resource {

	mime, _ := mimetype.DetectFile(localFilepath)

	modTime := time.Time{}

	if localFile, err := os.Open(localFilepath); err == nil {
		if stats, err := localFile.Stat(); err == nil {
			modTime = stats.ModTime()
		}
	}

	res := &Resource{
		ResourcePath:  resourcePath,
		LocalFilepath: localFilepath,
		Data:          data,
		MimeData:      mime,
		ModTime:       modTime,
	}

	project.Resources[resourcePath] = res
	return res
}

func (res *Resource) IsTexture() bool {
	_, isTexture := res.Data.(rl.Texture2D)
	return isTexture
}

func (res *Resource) Texture() rl.Texture2D {
	return res.Data.(rl.Texture2D)
}

func (res *Resource) IsGIF() bool {
	_, isGIF := res.Data.(*gif.GIF)
	return isGIF
}

func (res *Resource) GIF() *gif.GIF {
	return res.Data.(*gif.GIF)
}

func (res *Resource) IsAudio() bool {
	return strings.Contains(res.MimeData.String(), "audio")
}

// Audio is special in that there is no resource to be shared between Tasks like with Images, as each Task
// should have its own stream it manages to play back audio. So instead, the resource's Audio() function returns
// a brand new stream pointing to the audio file. The Task (or whatever uses the Stream) has to handle closing
// the Stream when it's deleted (which it does in the ReceiveMessage() function when it is informed that it is
// going to be deleted).
func (res *Resource) Audio() (beep.StreamSeekCloser, beep.Format, error) {

	var stream beep.StreamSeekCloser
	var format beep.Format
	var err error

	if res.IsAudio() {

		file, err := os.Open(res.LocalFilepath)
		if err != nil {
			log.Println("Could not open audio file: ", err.Error())
			currentProject.Log("Could not open audio file: %s", err.Error())
		} else {

			switch ext := res.MimeData.Extension(); ext {
			case ".wav":
				stream, format, err = wav.Decode(file)
			case ".flac":
				stream, format, err = flac.Decode(file)
			case ".ogg":
			case ".oga":
				stream, format, err = vorbis.Decode(file)
			case ".mp3":
				stream, format, err = mp3.Decode(file)
			}

			if err != nil {
				log.Println("Error decoding audio file: ", err.Error())
				currentProject.Log("Error decoding audio file: %s", err.Error())
			}

		}

	}

	return stream, format, err

}

func (res *Resource) Destroy() {

	if res.IsTexture() {
		rl.UnloadTexture(res.Texture())
	}
	// GIFs don't need to be disposed of directly here; the file handle was already Closed.
	// Audio streams are closed by the Task, as each Sound Task has its own stream.

	if res.Temporary {
		os.Remove(res.LocalFilepath)
	}

}
