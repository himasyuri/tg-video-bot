package telegram

import (
	"github.com/go-telegram/bot"
)

// RegisterHandlers registers the bot handlers.
func RegisterHandlers(b *bot.Bot, h *Handlers) {
	b.RegisterHandler(bot.HandlerTypeMessageText, "/start", bot.MatchTypeExact, h.StartHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/help", bot.MatchTypeExact, h.HelpHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "", bot.MatchTypePrefix, h.MessageHandler)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, "action_", bot.MatchTypePrefix, h.CallbackHandler)
}
