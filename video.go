package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
)

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

	type Stream struct {
		AspectRatio string `json:"display_aspect_ratio"`
	}
	type FFProbeOutput struct {
		Streams []Stream `json:"streams"`
	}

	var result FFProbeOutput
	if err := json.Unmarshal(output, &result); err != nil {
		return "", err
	}
	if len(result.Streams) == 0 {
		return "", nil
	}
	return result.Streams[0].AspectRatio, nil
}
