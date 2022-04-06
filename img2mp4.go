package img2mp4

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"log"
	"math"
	"os"
	"regexp"
	"strconv"

	ffmpeg "github.com/damaredayo/go-fluent-ffmpeg"
)

var intRegex = regexp.MustCompile(`\d+`)

type MP4Writer struct {
	ffmpeg  ffmpeg.Command
	output  string
	size    string
	fps     float64
	bitrate int

	imageBuffer []byte

	imageDirectory string
}

func New(output string, size string, fps float64, bitrate int, imageDirectory string) (*MP4Writer, error) {

	imageBytes := make([]byte, 0)

	// iterate over files in directory
	files, err := ioutil.ReadDir(imageDirectory)
	if err != nil {
		return nil, err
	}

	// get the name of all files into a slice
	fileNames := make([]string, len(files))
	for _, file := range files {
		// extract int from string, only get the first result
		intRes := intRegex.FindAllString(file.Name(), 1)
		if len(intRes) > 0 {
			// convert string to int
			intVal, err := strconv.Atoi(intRes[0])
			if err != nil {
				return nil, err
			}
			// add to slice, if there are duplicate numbers, this is user error and not my problem :)
			fileNames[intVal] = file.Name()
		}
	}

	// now the slice is in a good order, iterate over it and append the bytes
	for _, fileName := range fileNames {
		// open file
		file, err := os.Open(fmt.Sprintf("%s/%s", imageDirectory, fileName))
		if err != nil {
			return nil, err
		}
		// read file
		fileBytes, err := ioutil.ReadAll(file)
		if err != nil {
			return nil, err
		}
		// append bytes
		imageBytes = append(imageBytes, fileBytes...)
	}

	return &MP4Writer{
		output:         output,
		size:           size,
		fps:            fps,
		bitrate:        bitrate,
		imageDirectory: imageDirectory,
		imageBuffer:    imageBytes,
	}, nil
}

func (m *MP4Writer) FfmpegStart() error {

	imageBytesReader := bytes.NewReader(m.imageBuffer)

	pr, pw := io.Pipe()

	cmd := ffmpeg.NewCommand("ffmpeg").
		PipeInput(imageBytesReader).
		PipeOutput(pw).
		VideoCodec("libx264").
		InputOptions("-f image2pipe").
		VideoBitRate(m.bitrate).
		FrameRate(int(math.Floor(m.fps))).
		Resolution(m.size)

	go func() {
		defer pw.Close()
		err := cmd.Run()
		if err != nil {
			log.Printf("Ffmpeg Error: %v\n", err)
			return
		}
	}()

	b, err := ioutil.ReadAll(pr)
	if err != nil {
		log.Printf("Ffmpeg Pipe Error: %v\n", err)
		return err
	}

	err = ioutil.WriteFile(m.output, b, 0644)
	if err != nil {
		return err
	}

	log.Println("Ffmpeg finished, wrote file to:", m.output)

	return nil
}

// Set the length of the video in seconds, will calculate based on number of images in current struct.
func (w *MP4Writer) SetLength(length int) error {
	imgs, err := w.GetFiles()
	if err != nil {
		return err
	}

	w.fps = float64(len(imgs)) / float64(length)
	return nil
}

// Get the files in the directory defined in the struct.
func (w *MP4Writer) GetFiles() ([]fs.FileInfo, error) {
	// open directory
	files, err := ioutil.ReadDir(w.imageDirectory)
	if err != nil {
		return nil, err
	}
	// return count
	return files, nil
}
