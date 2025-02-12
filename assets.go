package main

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"strings"
)

func (cfg apiConfig) ensureAssetsDir() error {
	if _, err := os.Stat(cfg.assetsRoot); os.IsNotExist(err) {
		return os.Mkdir(cfg.assetsRoot, 0755)
	}
	return nil
}


func getAssetPath(mediaType string) string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		panic("failed to generate random bytes")
	}

	assetName := base64.RawURLEncoding.EncodeToString(bytes)
	return assetName + mediaTypeToExt(mediaType)
}

func mediaTypeToExt(mediaType string) string {
	parts := strings.Split(mediaType, "/")
	if len(parts) != 2 {
		return ".bin"
	}
	return "." + parts[1]
}

func isValidImageType(mediaType string) bool {
	switch mediaType {
		case "image/jpeg":
			return true
		case "image/png":
			return true
		default:
			return false
	}
}

func getVideoAspectRatio(filePath string) (string, error) {
	cmd := exec.Command("ffprobe", 
		"-v", "error", 
		"-print_format", "json", 
		"-show_streams", 
		filePath)
	
	var buffer bytes.Buffer
	cmd.Stdout = &buffer

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("ffprobe error: %v", err)
	}

	var output struct {
		Streams []struct {
			Width int `json:"width"`
			Height int `json:"height"`
		} `json:"streams"`
	}

	if err := json.Unmarshal(buffer.Bytes(), &output); err != nil {
		return "", fmt.Errorf("could not parse ffprobe output: %v", err)
	}

	if len(output.Streams) == 0 {
		return "", errors.New("no video streams found")
	}

	landscapeRatio := 16.0 / 9.0
	portraitRatio := 9.0 / 16.0
	videoRatio := float64(output.Streams[0].Width) / float64(output.Streams[0].Height)

	if math.Abs(videoRatio-landscapeRatio) < 0.1 {
		return "16:9", nil
	}
	if math.Abs(videoRatio-portraitRatio) < 0.1 {
		return "9:16", nil
	}
	return "other", nil
}

func processVideoForFastStart(filePath string) (string, error) {
	processedFilePath := filePath + ".processing"
	cmd := exec.Command("ffmpeg", 
		"-i", filePath, 
		"-c", "copy", 
		"-movflags", "faststart",
		"-f", "mp4",
		processedFilePath)
	
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("error processing video: %s, %v", stderr.String(), err)
	}

	fileInfo, err := os.Stat(processedFilePath)
	if err != nil {
		return "", fmt.Errorf("could not stat processed file: %v", err)
	}
	if fileInfo.Size() == 0 {
		return "", fmt.Errorf("processed file is empty")
	}

	return processedFilePath, nil
}