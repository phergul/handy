package main

import (
	"bytes"
	"fmt"
	"log"
	"math"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

func main() {
	if len(os.Args) < 4 {
		fmt.Printf("Usage: %s <input.mp4> <target_size_MB> <output.mp4>\n", os.Args[0])
		os.Exit(1)
	}

	input := os.Args[1]
	targetSizeMB, err := strconv.ParseFloat(os.Args[2], 64)
	if err != nil {
		log.Fatalf("Error converting target size (MB) to float: %v", err)
	}
	output := os.Args[3]

	if targetSizeMB <= 0 {
		log.Fatal("Target size must be greater than 0 MB")
	}

	duration, err := getDuration(input)
	if err != nil {
		log.Fatalf("Error getting duration: %v", err)
	}

	totalBitrate := (targetSizeMB * 8192) / duration
	audioBitrate := 128.0
	videoBitrate := math.Max(totalBitrate-audioBitrate, 100.0)

	fmt.Printf("Target: %.2f MB (%.1f sec)\n", targetSizeMB, duration)
	fmt.Printf("Video bitrate: %.0f kbps, Audio: %.0f kbps\n", videoBitrate, audioBitrate)

	pass1 := exec.Command("ffmpeg", "-y", "-i", input,
		"-b:v", fmt.Sprintf("%.0fk", videoBitrate),
		"-b:a", fmt.Sprintf("%.0fk", audioBitrate),
		"-c:v", "libx264", "-pass", "1", "-an", "-f", "mp4", os.DevNull,
	)
	pass2 := exec.Command("ffmpeg", "-y", "-i", input,
		"-b:v", fmt.Sprintf("%.0fk", videoBitrate),
		"-b:a", fmt.Sprintf("%.0fk", audioBitrate),
		"-c:v", "libx264", "-pass", "2", output,
	)

	pass1.Stdout = nil
	pass1.Stderr = nil
	pass2.Stdout = nil
	pass2.Stderr = nil

	fmt.Println("Running pass 1...")
	if err := pass1.Run(); err != nil {
		log.Fatalf("Pass 1 failed: %v", err)
	}

	fmt.Println("Running pass 2...")
	if err := pass2.Run(); err != nil {
		log.Fatalf("Pass 2 failed: %v", err)
	}

	fmt.Printf("Compression complete: %s\n", output)
}

func getDuration(filename string) (float64, error) {
	cmd := exec.Command("ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		filename,
	)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return 0, fmt.Errorf("ffprobe failed: %v", err)
	}

	duration, err := strconv.ParseFloat(strings.TrimSpace(out.String()), 64)
	if err != nil {
		return 0, fmt.Errorf("invalid duration: %v", err)
	}
	return duration, nil
}
