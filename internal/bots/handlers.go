package bots

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/manuelpepe/tincho/internal/tincho"
)

type Handlers struct {
	service *tincho.Service
	logger  *slog.Logger
}

func NewHandlers(logger *slog.Logger, service *tincho.Service) Handlers {
	return Handlers{service: service, logger: logger.With("component", "bots-handlers")}
}

func (h *Handlers) AddBot(w http.ResponseWriter, r *http.Request) {
	roomID := r.URL.Query().Get("room")
	if roomID == "" {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("missing room attribute"))
		return
	}
	difficulty := r.URL.Query().Get("difficulty")
	if difficulty == "" {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("missing difficulty attribute"))
		return
	}
	room, ok := h.service.GetRoom(roomID)
	if !ok {
		h.logger.Error("Error getting room index")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error getting room index"))
		return
	}
	player := tincho.NewPlayer(RandomBotName())
	newLogger := h.logger.With("player", player.ID)
	bot, err := NewBot(newLogger, room.Context, player, difficulty)
	if err != nil {
		h.logger.Error("Error creating bot", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error creating bot"))
		return
	}
	go func() {
		if err := bot.Start(); err != nil {
			h.logger.Error("Error with bot: %s", err)
		}
		// TODO: If bot fails, broadcasts are stuck because noone is reading from the updates channel.
		// probably should tear down room and remove players or fallback to some known behaviour with an
		// error sent to all players.
	}()
	if err := h.service.JoinRoom(roomID, player, h.service.GetRoomPassword(roomID)); err != nil {
		h.logger.Error("Error joining room", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error joining room"))
		return
	}
	h.logger.Info(fmt.Sprintf("Bot %s joined room %s", player.ID, roomID))
}
