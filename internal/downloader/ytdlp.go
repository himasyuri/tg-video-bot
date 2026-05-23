package downloader

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// YTDLP is a wrapper for the yt-dlp command-line tool.
type YTDLP struct {
	OutputDir          string
	CookiesPath        string
	CookiesFromBrowser string
}

// NewYTDLP creates a new YTDLP downloader.
func NewYTDLP(outputDir, cookiesPath, cookiesFromBrowser string) *YTDLP {
	return &YTDLP{
		OutputDir:          outputDir,
		CookiesPath:        cookiesPath,
		CookiesFromBrowser: cookiesFromBrowser,
	}
}

// ytdlpMetadata represents the subset of JSON output from yt-dlp -J.
type ytdlpMetadata struct {
	ID       string  `json:"id"`
	Title    string  `json:"title"`
	Duration float64 `json:"duration"`
	Webpage  string  `json:"webpage_url_domain"`
	Ext      string  `json:"ext"`
}

// Download fetches the video using yt-dlp.
func (y *YTDLP) Download(ctx context.Context, url string) (*Result, error) {
	// Ensure output directory exists
	if err := os.MkdirAll(y.OutputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	// 1. Get metadata
	args := []string{"-J", "--no-playlist", url}
	if y.CookiesFromBrowser != "" {
		args = append(args, "--cookies-from-browser", y.CookiesFromBrowser)
	} else if y.CookiesPath != "" {
		if _, err := os.Stat(y.CookiesPath); err == nil {
			args = append(args, "--cookies", y.CookiesPath)
		}
	}

	cmd := exec.CommandContext(ctx, "yt-dlp", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to get metadata: %w, output: %s", err, string(output))
	}

	var metadata ytdlpMetadata
	if err := json.Unmarshal(output, &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse metadata: %w", err)
	}

	// 2. Download and merge to mp4
	outputPathTemplate := filepath.Join(y.OutputDir, "%(id)s.%(ext)s")

	downloadArgs := []string{
		"-f", "bestvideo[ext=mp4]+bestaudio[ext=m4a]/best[ext=mp4]/best",
		"--merge-output-format", "mp4",
		"-o", outputPathTemplate,
		"--no-playlist",
		url,
	}
	if y.CookiesFromBrowser != "" {
		downloadArgs = append(downloadArgs, "--cookies-from-browser", y.CookiesFromBrowser)
	} else if y.CookiesPath != "" {
		if _, err := os.Stat(y.CookiesPath); err == nil {
			downloadArgs = append(downloadArgs, "--cookies", y.CookiesPath)
		}
	}

	downloadCmd := exec.CommandContext(ctx, "yt-dlp", downloadArgs...)

	if output, err := downloadCmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("failed to download video: %w, output: %s", err, string(output))
	}
	// The final file should be <id>.mp4 due to merge-output-format
	finalPath := filepath.Join(y.OutputDir, metadata.ID+".mp4")

	return &Result{
		FilePath: finalPath,
		Title:    metadata.Title,
		Duration: int(metadata.Duration),
		Platform: metadata.Webpage,
		Ext:      "mp4",
	}, nil
}
