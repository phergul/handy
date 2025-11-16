package main

import (
	"archive/zip"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

var (
	zipName       = "mp4_compressor.zip"
	buildEmbedded = flag.Bool("embedded", false, "Build single-binary embedded installer instead of zip")
)

func main() {
	flag.Parse()

	wd, _ := os.Getwd()

	if err := buildCompressor(); err != nil {
		log.Printf("Warning: Failed to build mp4_compress.exe: %v\n", err)
	}

	if *buildEmbedded {
		if err := buildEmbeddedInstaller(); err != nil {
			log.Fatalf("ERROR: Failed to build embedded installer: %v\n", err)
		}
		return
	}

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


**Two installation options:**

### Option 1: Single-Binary Installer (Recommended)
Download ` + "`mp4_compress_installer.exe`" + `and run it. It will:
- Check for and install ffmpeg if needed
- Install the compressor
- Register the right-click context menu

### Option 2: ZIP Install
Download the zip file, extract it, and run ` + "`install.ps1`" + `with PowerShell.

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

func buildCompressor() error {
	compressorSource := filepath.Join("internal", "mp4_compress.go")
	compressorBinary := filepath.Join("internal", "mp4_compress.exe")

	if _, err := os.Stat(compressorSource); os.IsNotExist(err) {
		return fmt.Errorf("source file not found: %s", compressorSource)
	}

	fmt.Print("Building mp4_compress.exe...")
	cmd := exec.Command("go", "build", "-o", compressorBinary, compressorSource)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(" FAILED")
		fmt.Println(string(output))
		return err
	}
	fmt.Println(" DONE")
	return nil
}

func buildEmbeddedInstaller() error {
	fmt.Println("Building single-binary embedded installer...")

	if _, err := os.Stat("go.mod"); os.IsNotExist(err) {
		fmt.Print("Initializing Go module...")
		cmd := exec.Command("go", "mod", "init", "mp4_compress_installer")
		if err := cmd.Run(); err != nil {
			fmt.Println(" FAILED")
			return err
		}
		fmt.Println(" DONE")
	}

	fmt.Print("Downloading dependencies...")
	cmd := exec.Command("go", "mod", "tidy")
	if err := cmd.Run(); err != nil {
		fmt.Println(" FAILED")
		return err
	}
	fmt.Println(" DONE")

	fmt.Print("Building installer...")
	cmd = exec.Command("go", "build", "-o", "mp4_compress_installer.exe", "installer.go")
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(" FAILED")
		fmt.Println(string(output))
		return err
	}
	fmt.Println(" DONE")

	if _, err := os.Stat("mp4_compress_installer.exe"); err == nil {
		fmt.Println()
		fmt.Println("Success! Installer created: mp4_compress_installer.exe")
		fmt.Println()
		fmt.Println("Run mp4_compress_installer.exe to install the video compressor.")
	} else {
		return fmt.Errorf("installer binary not found after build")
	}

	return nil
}
