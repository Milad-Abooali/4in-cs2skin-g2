package handlers

import (
	"github.com/Milad-Abooali/4in-cs2skin-g2/src/internal/events"
	"github.com/Milad-Abooali/4in-cs2skin-g2/src/internal/models"
)

type CrashHistory struct {
	data []float64
	size int
}

var History = NewCrashHistory(50)

func NewCrashHistory(limit int) *CrashHistory {
	return &CrashHistory{
		data: make([]float64, 0, limit),
		size: limit,
	}
}

func (h *CrashHistory) Add(value float64) {
	if len(h.data) >= h.size {
		h.data = h.data[1:]
	}
	h.data = append(h.data, value)
}

func (h *CrashHistory) GetAll() []float64 {
	return h.data
}

func GetHistory(data map[string]interface{}) (models.HandlerOK, models.HandlerError) {
	var (
		errR models.HandlerError
		resR models.HandlerOK
	)

	events.Emit("all", "history", History.GetAll())

	// Success
	resR.Type = "getHistory"
	resR.Data = nil
	return resR, errR
}
