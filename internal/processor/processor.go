package processor

import "context"

// Options contains parameters for video processing.
type Options struct {
	StartTime  string // e.g., "00:00:10"
	Duration   string // e.g., "00:00:30"
	Format     string // e.g., "mp4", "mp3", "mkv"
	Compress   bool
	VideoCodec string // e.g., "libx264", "libx265", "libvpx-vp9"
	AudioCodec string // e.g., "aac", "libmp3lame", "opus"
}

// Result contains information about the processed video.
type Result struct {
	FilePath string
	Ext      string
}

// Processor is an interface for video processing logic.
type Processor interface {
	Process(ctx context.Context, inputPath string, opts Options) (*Result, error)
	GetAudio(ctx context.Context, inputPath string) (*Result, error)
}
