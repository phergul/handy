package main

import (
	"archive/zip"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

var (
	zipName = "mp4_compressor.zip"
)

func main() {
	wd, _ := os.Getwd()

	timestamp := time.Now().Format("20060102_150405")
	zipName = fmt.Sprintf("mp4_compressor_%s.zip", timestamp)
	zipPath := filepath.Join(wd, zipName)

	fmt.Print("Packaging mp4 compressor...")

	zipFile, err := os.Create(zipPath)
	if err != nil {
		log.Fatalf("ERROR: Failed to create zip file: %v\n", err)
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	if err := addFileToZip(zipWriter, "install.ps1", "install.ps1"); err != nil {
		log.Fatalf("\nERROR: Failed to add install.ps1: %v\n", err)
	}

	readme := `# MP4 Video Compressor - Installation

A simple right-click video compression tool for Windows.

## Installation

1. Right-click install.ps1 and select "Run with PowerShell"
2. Accept the prompt for admin privileges
3. The installer will:
   - Install ffmpeg (if needed)
   - Set up the compression tool
   - Add right-click context menu for .mp4 files

## Usage

After installation, right-click any .mp4 file and select "Compress Video"

## Uninstall

Run: %LOCALAPPDATA%\mp4_compress\uninstall.ps1
`
	if err := addStringToZip(zipWriter, "README.txt", readme); err != nil {
		log.Fatalf("\nERROR: Failed to add README.txt: %v\n", err)
	}

	if err := addDirToZip(zipWriter, "internal", "internal"); err != nil {
		log.Fatalf("\nERROR: Failed to add internal directory: %v\n", err)
	}

	fmt.Println(" DONE")

	fmt.Printf("Created at: %s\n", zipPath)
	fmt.Println()
}

func addFileToZip(zipWriter *zip.Writer, filePath, zipPath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return err
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	header.Name = zipPath
	header.Method = zip.Deflate

	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return err
	}

	_, err = io.Copy(writer, file)
	return err
}

func addStringToZip(zipWriter *zip.Writer, zipPath, content string) error {
	writer, err := zipWriter.Create(zipPath)
	if err != nil {
		return err
	}
	_, err = writer.Write([]byte(content))
	return err
}

func addDirToZip(zipWriter *zip.Writer, dirPath, zipPrefix string) error {
	return filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(dirPath, path)
		if err != nil {
			return err
		}

		zipPath := filepath.Join(zipPrefix, relPath)
		return addFileToZip(zipWriter, path, zipPath)
	})
}
