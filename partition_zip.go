package main

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	MaxPartitionSize = 1024 * 1024 * 1000 // 1 GB
)

var (
	zipDir string
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run partition_zip.go <directory_to_partition> <output_zip_directory>")
		return
	}

	startTime := time.Now()

	dir := os.Args[1]
	zipDir = os.Args[2]
	info, err := os.Stat(dir)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	if !info.IsDir() {
		fmt.Println("Provided path is not a directory")
		return
	}

	partitions, err := partitionDirectory(dir)
	if err != nil {
		fmt.Println("Error partitioning directory:", err)
		return
	}

	// for i, partition := range partitions {
	// 	fmt.Printf("Partition %d:\n", i+1)
	// 	for _, file := range partition {
	// 		fmt.Println("  ", file)
	// 	}
	// }

	fmt.Printf("Total partitions created: %d\n", len(partitions))
	err = zipPartitions(partitions)
	if err != nil {
		fmt.Println("Error zipping partitions:", err)
		return
	}
	fmt.Println("Zipping completed successfully.")

	elapsed := time.Since(startTime)
	fmt.Printf("Total time taken: %s\n", elapsed)
}

func partitionDirectory(dir string) ([][]string, error) {
	var partitions [][]string
	var currentPartition []string
	var currentSize int64

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			fileSize := info.Size()
			if currentSize+fileSize > MaxPartitionSize {
				partitions = append(partitions, currentPartition)
				currentPartition = []string{}
				currentSize = 0
			}
			currentPartition = append(currentPartition, path)
			currentSize += fileSize
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(currentPartition) > 0 {
		partitions = append(partitions, currentPartition)
	}
	return partitions, nil
}

func zipPartitions(partitions [][]string) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(partitions))

	for i, partition := range partitions {
		wg.Add(1)
		go func(i int, partition []string) {
			defer wg.Done()
			zipFileName := fmt.Sprintf("partition_%d.zip", i+1)
			zipFile, err := os.Create(filepath.Join(zipDir, zipFileName))
			if err != nil {
				errChan <- err
				return
			}
			defer zipFile.Close()

			zipWriter := zip.NewWriter(zipFile)
			defer zipWriter.Close()

			for _, filePath := range partition {
				err := addFileToZip(zipWriter, filePath)
				if err != nil {
					errChan <- err
				}
			}
			fmt.Printf("Created zip file: %s\n", zipFileName)
		}(i, partition)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		fmt.Println("Error zipping partition:", err)
	}
	return nil
}

func addFileToZip(zipWriter *zip.Writer, filePath string) error {
	fileToZip, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer fileToZip.Close()

	info, err := fileToZip.Stat()
	if err != nil {
		return err
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	header.Name = filepath.Base(filePath)
	header.Method = zip.Deflate

	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return err
	}

	_, err = io.Copy(writer, fileToZip)
	return err
}
