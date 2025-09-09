package handlers

import (
	"github.com/Milad-Abooali/4in-cs2skin-g2/src/internal/models"
	"time"
)

// Ping - Handler
func Ping(data map[string]interface{}) (models.HandlerOK, models.HandlerError) {
	var (
		errR models.HandlerError
		resR models.HandlerOK
	)

	NextGame(1)

	// Success
	resR.Type = "ping"
	resR.Data = time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	return resR, errR
}
