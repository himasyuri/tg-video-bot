package processor

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// FFmpeg is a wrapper for the ffmpeg command-line tool.
type FFmpeg struct {
	OutputDir string
}

// NewFFmpeg creates a new FFmpeg processor.
func NewFFmpeg(outputDir string) *FFmpeg {
	return &FFmpeg{OutputDir: outputDir}
}

// Process handles re-encoding, cutting, and compression.
func (f *FFmpeg) Process(ctx context.Context, inputPath string, opts Options) (*Result, error) {
	if err := os.MkdirAll(f.OutputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	ext := opts.Format
	if ext == "" {
		ext = "mp4"
	}

	outputFilename := fmt.Sprintf("processed_%d.%s", time.Now().UnixNano(), ext)
	outputPath := filepath.Join(f.OutputDir, outputFilename)

	args := []string{"-i", inputPath}

	// Cutting logic
	if opts.StartTime != "" {
		args = append(args, "-ss", opts.StartTime)
	}
	if opts.Duration != "" {
		args = append(args, "-t", opts.Duration)
	}

	// Compression/Encoding logic
	vCodec := opts.VideoCodec
	aCodec := opts.AudioCodec

	if opts.Compress {
		if vCodec == "" {
			vCodec = "libx264"
		}
		if aCodec == "" {
			aCodec = "aac"
		}
		args = append(args, "-c:v", vCodec, "-crf", "28", "-preset", "faster", "-c:a", aCodec, "-b:a", "128k")
	} else {
		// Default or custom encoding
		if vCodec != "" {
			args = append(args, "-c:v", vCodec)
		} else if strings.ToLower(ext) == "mp4" {
			args = append(args, "-c:v", "libx264", "-preset", "veryfast")
		}

		if aCodec != "" {
			args = append(args, "-c:a", aCodec)
		} else if strings.ToLower(ext) == "mp4" {
			args = append(args, "-c:a", "aac")
		}
	}

	args = append(args, "-y", outputPath)

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("ffmpeg process failed: %w, output: %s", err, string(output))
	}

	return &Result{
		FilePath: outputPath,
		Ext:      ext,
	}, nil
}

// GetAudio extracts only the audio from the video.
func (f *FFmpeg) GetAudio(ctx context.Context, inputPath string) (*Result, error) {
	if err := os.MkdirAll(f.OutputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	outputFilename := fmt.Sprintf("audio_%d.mp3", time.Now().UnixNano())
	outputPath := filepath.Join(f.OutputDir, outputFilename)

	// Extract audio to mp3
	cmd := exec.CommandContext(ctx, "ffmpeg", "-i", inputPath, "-vn", "-acodec", "libmp3lame", "-q:a", "2", "-y", outputPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("ffmpeg audio extraction failed: %w, output: %s", err, string(output))
	}

	return &Result{
		FilePath: outputPath,
		Ext:      "mp3",
	}, nil
}
