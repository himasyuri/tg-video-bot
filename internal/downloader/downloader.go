package downloader

import "context"

// Result contains information about the downloaded video.
type Result struct {
	FilePath string
	Title    string
	Duration int // in seconds
	Platform string
	Ext      string
}

// Downloader is an interface for video downloading logic.
type Downloader interface {
	Download(ctx context.Context, url string) (*Result, error)
}
