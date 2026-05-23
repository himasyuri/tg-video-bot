package telegram

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/himasyuri/light-video-tgbot/internal/downloader"
	"github.com/himasyuri/light-video-tgbot/internal/processor"
	"github.com/himasyuri/light-video-tgbot/internal/storage/cache"
)

type Handlers struct {
	downloader downloader.Downloader
	processor  processor.Processor
	cache      *cache.Cache
	token      string
	tmpDir     string
}

func NewHandlers(d downloader.Downloader, p processor.Processor, c *cache.Cache, token string, tmpDir string) *Handlers {
	return &Handlers{
		downloader: d,
		processor:  p,
		cache:      c,
		token:      token,
		tmpDir:     tmpDir,
	}
}

func (h *Handlers) StartHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "Hello! I am Light Video Bot. Send me a link to a video to get started.",
	})
}

func (h *Handlers) MessageHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	chatID := update.Message.Chat.ID
	stats, _ := h.cache.GetUserStats(chatID)

	// Handle conversation states (multi-step commands)
	switch stats.ConversationState {
	case cache.StateAwaitingStartTime:
		h.handleStartTime(ctx, b, update, stats)
		return
	case cache.StateAwaitingEndTime:
		h.handleEndTime(ctx, b, update, stats)
		return
	}

	if update.Message.Video != nil {
		h.handleVideo(ctx, b, update)
		return
	}

	if update.Message.Text == "" {
		return
	}

	text := update.Message.Text
	if strings.HasPrefix(text, "http") {
		h.handleDownload(ctx, b, update)
		return
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "Please send me a valid video link or a video file.",
	})
}

func (h *Handlers) handleVideo(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID := update.Message.Chat.ID
	video := update.Message.Video

	// Check rate limit
	if !h.cache.CheckRateLimit(chatID) {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "Rate limit exceeded.",
		})
		return
	}

	// Store FileID in cache
	stats, _ := h.cache.GetUserStats(chatID)
	stats.PendingFileID = video.FileID
	stats.PendingURL = "" // Clear URL if video is sent
	h.cache.SetUserStats(chatID, stats)

	// Show processing options
	kb := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "Extract Audio", CallbackData: "action_audio"},
				{Text: "Re-encode (H.265)", CallbackData: "action_reencode"},
			},
			{
				{Text: "Compress", CallbackData: "action_compress"},
				{Text: "Get Piece", CallbackData: "action_cut"},
			},
		},
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        "What would you like to do with your uploaded video?",
		ReplyMarkup: kb,
	})
}

func (h *Handlers) handleDownload(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID := update.Message.Chat.ID
	url := update.Message.Text

	// Check rate limit: 5 requests per 3 hours
	if !h.cache.CheckRateLimit(chatID) {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "Rate limit exceeded. You can only make 5 requests every 3 hours.",
		})
		return
	}

	// Store URL in cache for subsequent callback
	stats, _ := h.cache.GetUserStats(chatID)
	stats.PendingURL = url
	h.cache.SetUserStats(chatID, stats)

	// Show processing options
	kb := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "Only Download", CallbackData: "action_download"},
				{Text: "Get Audio", CallbackData: "action_audio"},
			},
			{
				{Text: "Cut/Clip", CallbackData: "action_cut"},
				{Text: "Re-encode (H.265)", CallbackData: "action_reencode"},
			},
		},
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        "What would you like to do with this video?",
		ReplyMarkup: kb,
	})
}

func (h *Handlers) CallbackHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery == nil || update.CallbackQuery.Message.Message == nil {
		return
	}

	chatID := update.CallbackQuery.Message.Message.Chat.ID
	data := update.CallbackQuery.Data

	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
	})

	stats, found := h.cache.GetUserStats(chatID)
	if !found || (stats.PendingURL == "" && stats.PendingFileID == "") {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "Session expired. Please send the link or video again.",
		})
		return
	}

	url := stats.PendingURL
	fileID := stats.PendingFileID

	switch data {
	case "action_download":
		stats.PendingURL = ""
		stats.PendingFileID = ""
		h.cache.SetUserStats(chatID, stats)
		h.processRequest(ctx, b, chatID, url, fileID, processor.Options{})
	case "action_audio":
		stats.PendingURL = ""
		stats.PendingFileID = ""
		h.cache.SetUserStats(chatID, stats)
		h.processRequest(ctx, b, chatID, url, fileID, processor.Options{Format: "mp3"})
	case "action_reencode":
		stats.PendingURL = ""
		stats.PendingFileID = ""
		h.cache.SetUserStats(chatID, stats)
		h.processRequest(ctx, b, chatID, url, fileID, processor.Options{VideoCodec: "libx265"})
	case "action_compress":
		stats.PendingURL = ""
		stats.PendingFileID = ""
		h.cache.SetUserStats(chatID, stats)
		h.processRequest(ctx, b, chatID, url, fileID, processor.Options{Compress: true})
	case "action_cut":
		stats.ConversationState = cache.StateAwaitingStartTime
		h.cache.SetUserStats(chatID, stats)
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "Please send the START time (e.g., 1:30 or 00:01:30):",
		})
	}
}

func (h *Handlers) handleStartTime(ctx context.Context, b *bot.Bot, update *models.Update, stats cache.UserStats) {
	chatID := update.Message.Chat.ID
	stats.StartTime = update.Message.Text
	stats.ConversationState = cache.StateAwaitingEndTime
	h.cache.SetUserStats(chatID, stats)

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   "Great! Now send the END time (e.g., 1:40 or 00:01:40):",
	})
}

func (h *Handlers) handleEndTime(ctx context.Context, b *bot.Bot, update *models.Update, stats cache.UserStats) {
	chatID := update.Message.Chat.ID
	endTime := update.Message.Text
	startTime := stats.StartTime

	// Reset state
	url := stats.PendingURL
	fileID := stats.PendingFileID
	stats.ConversationState = cache.StateNone
	stats.PendingURL = ""
	stats.PendingFileID = ""
	stats.StartTime = ""
	h.cache.SetUserStats(chatID, stats)

	duration, err := h.calculateDuration(startTime, endTime)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   fmt.Sprintf("Invalid time format or duration: %v. Please start over.", err),
		})
		return
	}

	h.processRequest(ctx, b, chatID, url, fileID, processor.Options{
		StartTime: startTime,
		Duration:  fmt.Sprintf("%.2f", duration),
	})
}

func (h *Handlers) calculateDuration(start, end string) (float64, error) {
	s, err := h.parseTimeToSeconds(start)
	if err != nil {
		return 0, fmt.Errorf("invalid start time: %w", err)
	}
	e, err := h.parseTimeToSeconds(end)
	if err != nil {
		return 0, fmt.Errorf("invalid end time: %w", err)
	}

	if e <= s {
		return 0, fmt.Errorf("end time must be greater than start time")
	}

	return e - s, nil
}

func (h *Handlers) parseTimeToSeconds(t string) (float64, error) {
	t = strings.TrimSpace(t)
	parts := strings.Split(t, ":")
	var seconds float64

	switch len(parts) {
	case 1: // ss
		fmt.Sscanf(parts[0], "%f", &seconds)
	case 2: // mm:ss
		var m, s float64
		fmt.Sscanf(parts[0], "%f", &m)
		fmt.Sscanf(parts[1], "%f", &s)
		seconds = m*60 + s
	case 3: // hh:mm:ss
		var h, m, s float64
		fmt.Sscanf(parts[0], "%f", &h)
		fmt.Sscanf(parts[1], "%f", &m)
		fmt.Sscanf(parts[2], "%f", &s)
		seconds = h*3600 + m*60 + s
	default:
		return 0, fmt.Errorf("invalid format")
	}
	return seconds, nil
}

func (h *Handlers) processRequest(ctx context.Context, b *bot.Bot, chatID int64, url, fileID string, opts processor.Options) {
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   "Processing, please wait...",
	})

	var inputPath string
	var title string
	var duration int
	var isAudio bool = opts.Format == "mp3"

	// 1. Get input file
	if url != "" {
		dlResult, err := h.downloader.Download(ctx, url)
		if err != nil {
			b.SendMessage(ctx, &bot.SendMessageParams{ChatID: chatID, Text: fmt.Sprintf("Download failed: %v", err)})
			return
		}
		inputPath = dlResult.FilePath
		title = dlResult.Title
		duration = dlResult.Duration
		defer os.Remove(inputPath)
	} else if fileID != "" {
		path, err := h.downloadTelegramFile(ctx, b, fileID)
		if err != nil {
			b.SendMessage(ctx, &bot.SendMessageParams{ChatID: chatID, Text: fmt.Sprintf("Failed to get file from TG: %v", err)})
			return
		}
		inputPath = path
		title = "uploaded_video"
		duration = 0
		defer os.Remove(inputPath)
	}

	// 2. Process
	var finalPath string
	if isAudio {
		res, err := h.processor.GetAudio(ctx, inputPath)
		if err != nil {
			b.SendMessage(ctx, &bot.SendMessageParams{ChatID: chatID, Text: fmt.Sprintf("Processing failed: %v", err)})
			return
		}
		finalPath = res.FilePath
		defer os.Remove(finalPath)
		h.sendAudioResult(ctx, b, chatID, finalPath, title)
	} else {
		res, err := h.processor.Process(ctx, inputPath, opts)
		if err != nil {
			b.SendMessage(ctx, &bot.SendMessageParams{ChatID: chatID, Text: fmt.Sprintf("Processing failed: %v", err)})
			return
		}
		finalPath = res.FilePath
		defer os.Remove(finalPath)
		h.sendVideoResult(ctx, b, chatID, finalPath, title, duration)
	}
}

func (h *Handlers) downloadTelegramFile(ctx context.Context, b *bot.Bot, fileID string) (string, error) {
	file, err := b.GetFile(ctx, &bot.GetFileParams{FileID: fileID})
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", h.token, file.FilePath)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bad status: %s", resp.Status)
	}

	outPath := filepath.Join(h.tmpDir, fileID+".mp4")
	out, err := os.Create(outPath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return outPath, err
}

func (h *Handlers) sendAudioResult(ctx context.Context, b *bot.Bot, chatID int64, filePath, title string) {
	file, err := os.Open(filePath)
	if err != nil {
		return
	}
	defer file.Close()
	b.SendAudio(ctx, &bot.SendAudioParams{
		ChatID: chatID,
		Audio:  &models.InputFileUpload{Data: file, Filename: title + ".mp3"},
		Title:  title,
	})
}

func (h *Handlers) sendVideoResult(ctx context.Context, b *bot.Bot, chatID int64, filePath string, title string, duration int) {
	file, err := os.Open(filePath)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   fmt.Sprintf("Failed to open file: %v", err),
		})
		return
	}
	defer file.Close()

	if duration > 0 && duration < 300 {
		b.SendVideo(ctx, &bot.SendVideoParams{
			ChatID:  chatID,
			Video:   &models.InputFileUpload{Data: file, Filename: title + ".mp4"},
			Caption: title,
		})
	} else {
		b.SendDocument(ctx, &bot.SendDocumentParams{
			ChatID:   chatID,
			Document: &models.InputFileUpload{Data: file, Filename: title + ".mp4"},
			Caption:  title,
		})
	}
}
