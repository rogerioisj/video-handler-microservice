package converter

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"time"
)

type VideoConverter struct {
}

func NewVideoConverter() *VideoConverter {
	return &VideoConverter{}
}

type VideoTask struct {
	VideoID int    `json:"video_id"`
	Path    string `json:"path"`
}

func (vc *VideoConverter) Handle(msg []byte) {
	var task VideoTask

	err := json.Unmarshal(msg, &task)
	if err != nil {
		vc.logError(task, "Failed to unmarshal json", err)
		return
	}

	err = vc.processVideo(&task)

	if err != nil {
		vc.logError(task, "Failed to process video", err)
		return
	}
}

func (vc *VideoConverter) processVideo(task *VideoTask) error {
	mergedFile := filepath.Join(task.Path, "merged.mp4")
	mpegDashPath := filepath.Join(task.Path, "mpeg-dash")

	slog.Info("Merging chunks", slog.String("mergedFile", mergedFile))

	err := vc.mergeChunks(task.Path, mergedFile)

	if err != nil {
		vc.logError(*task, "Failed to merge chunks", err)
		return fmt.Errorf("Failed to merge chunks: %v", err)
	}

	slog.Info("Creating mpeg-dash directory", slog.String("mpegDashPath", mpegDashPath))
	err = os.MkdirAll(mpegDashPath, os.ModeAppend)

	if err != nil {
		vc.logError(*task, "Failed to create directory", err)
		return fmt.Errorf("Failed to create directory: %v", err)
	}

	slog.Info("Converting video to mpeg-dash", slog.String("path", task.Path))
	ffmpegCmd := exec.Command(
		"ffmpeg", "-i", mergedFile,
		"-f", "dash",
		filepath.Join(mpegDashPath, "output.mpd"),
	)

	output, err := ffmpegCmd.CombinedOutput()

	if err != nil {
		vc.logError(*task, "Failed to convert video to mpeg"+string(output), err)
		return fmt.Errorf("Failed to convert video to mpeg: %v", err)
	}

	slog.Info("Finished converting video to mpeg", slog.String("path", mpegDashPath))

	err = os.Remove(mergedFile)

	if err != nil {
		vc.logError(*task, "Failed to remove merged file", err)
		return fmt.Errorf("Failed to remove merged file: %v", err)
	}

	return nil
}

func (vc *VideoConverter) logError(task VideoTask, message string, receivedError error) {
	errorData := map[string]any{
		"video_id": task.VideoID,
		"error":    message,
		"details":  receivedError.Error(),
		"time":     time.Now(),
	}

	serializedError, err := json.Marshal(errorData)

	if err != nil {
		panic(err)
	}

	slog.Error("Processing error", slog.String("error_details", string(serializedError)))
}

func (vc *VideoConverter) extractNumber(fileName string) int {
	//fmt.Printf("extracting number for %v \n", fileName)
	regex := regexp.MustCompile(`\d+`)
	stringNumber := regex.FindString(filepath.Base(fileName))
	number, err := strconv.Atoi(stringNumber)

	if err != nil {
		return -1
	}

	return number
}

func (vc *VideoConverter) mergeChunks(inputDir, outputFile string) error {
	fmt.Println("Starting process")
	chunks, err := filepath.Glob(filepath.Join(inputDir, "*.chunk"))

	if err != nil {
		return fmt.Errorf("Failed to find chunks: %v", err)
	}

	fmt.Printf("Found %v chunks\n", len(chunks))
	sort.Slice(chunks, func(i, j int) bool {
		return vc.extractNumber(chunks[i]) < vc.extractNumber(chunks[j])
	})

	output, err := os.Create(outputFile)

	if err != nil {
		return fmt.Errorf("Failed to create output file: %v", err)
	}

	defer output.Close()

	for _, chunk := range chunks {
		fmt.Printf("Unifying chunk %s \n", chunk)
		input, err := os.Open(chunk)

		if err != nil {
			return fmt.Errorf("Failed to open input file: %v", err)
		}

		_, err = io.Copy(output, input)

		if err != nil {
			return fmt.Errorf("Failed to read input file: %v", err)
		}

		input.Close()
	}

	return nil
}
