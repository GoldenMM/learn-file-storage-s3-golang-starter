package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
)

// Structs to unmarshal the ffprobe output
type Stream struct {
	AspectRatio string `json:"display_aspect_ratio"`
}
type FFProbeOutput struct {
	Streams []Stream `json:"streams"`
}

/*
getVideoAspectRatio uses ffprobe to get the aspect ratio of a video file
It returns the aspect ratio as a string (e.g., "16:9", "9:16") or an error if it fails
to execute ffprobe or parse the output.
*/
func getVideoAspectRatio(filePath string) (string, error) {
	// Use ffprobe to get the video aspect ratio

	cmd := exec.Command(
		"ffprobe",
		"-v", "error",
		"-print_format", "json",
		"-show_streams",
		filePath,
	)
	cmd.Stdout = &bytes.Buffer{}
	cmd.Stderr = &bytes.Buffer{}

	err := cmd.Run()
	if err != nil {
		fmt.Println("ffprobe error:", err)
		fmt.Println("ffprobe stderr:", cmd.Stderr.(*bytes.Buffer).String())
		return "", err
	}

	// Unmarshal the output to get the aspect ratio
	output := cmd.Stdout.(*bytes.Buffer).Bytes()

	var result FFProbeOutput
	if err := json.Unmarshal(output, &result); err != nil {
		return "", err
	}
	if len(result.Streams) == 0 {
		return "", nil
	}
	return result.Streams[0].AspectRatio, nil
}

/*
processVideoForFastStart uses ffmpeg to process a video file for fast start
It copies the video stream and sets the movflags to faststart
It returns the path to the processed video file or an error if it fails
*/
func processVideoForFastStart(filePath string) (string, error) {

	outputFilePath := filePath + ".processing"

	// Use ffmpeg to process the video for fast start
	cmd := exec.Command(
		"ffmpeg",
		"-i", filePath,
		"-c", "copy",
		"-movflags", "faststart",
		"-f", "mp4",
		outputFilePath,
	)
	cmd.Stdout = &bytes.Buffer{}
	cmd.Stderr = &bytes.Buffer{}

	err := cmd.Run()
	if err != nil {
		fmt.Println("ffprobe error:", err)
		fmt.Println("ffprobe stderr:", cmd.Stderr.(*bytes.Buffer).String())
		return "", err
	}
	// Check if the output file was created
	if _, err := os.Stat(outputFilePath); err != nil {
		fmt.Println("Output file stat error:", err)
		return "", err
	}
	return outputFilePath, nil
}
