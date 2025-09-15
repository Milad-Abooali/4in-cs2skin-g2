package handlers

import (
	"github.com/Milad-Abooali/4in-cs2skin-g2/src/internal/events"
	"github.com/Milad-Abooali/4in-cs2skin-g2/src/internal/models"
	"time"
)

// Ping - Handler
func Ping(_ map[string]interface{}) (models.HandlerOK, models.HandlerError) {
	var (
		errR models.HandlerError
		resR models.HandlerOK
	)

	events.Emit("all", "history", History.GetAll())
	events.Emit("all", "leaderboard", Leaderboard.GetAll())
	events.Emit("all", "liveBets", LiveBets)
	events.Emit("all", "liveGame", LiveGame)

	// Success
	resR.Type = "ping"
	resR.Data = time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	return resR, errR
}
