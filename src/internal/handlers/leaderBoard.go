package handlers

import (
	"sync"

	"github.com/Milad-Abooali/4in-cs2skin-g2/src/internal/events"
	"github.com/Milad-Abooali/4in-cs2skin-g2/src/internal/models"
)

// CrashLeaderboard keeps track of last N paid bets
type CrashLeaderboard struct {
	data []models.Bet
	size int
	mu   sync.Mutex
}

// Leaderboard Global instance (limit 20 items)
var Leaderboard = NewCrashLeaderboard(20)

// NewCrashLeaderboard Constructor
func NewCrashLeaderboard(limit int) *CrashLeaderboard {
	return &CrashLeaderboard{
		data: make([]models.Bet, 0, limit),
		size: limit,
	}
}

// Add a new paid bet to leaderboard
func (lb *CrashLeaderboard) Add(bet models.Bet) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	if len(lb.data) >= lb.size {
		lb.data = lb.data[1:] // remove oldest
	}
	lb.data = append(lb.data, bet)

	// Emit event whenever leaderboard changes
	events.Emit("all", "leaderboard", lb.GetAll())
}

// GetAll returns a snapshot of leaderboard
func (lb *CrashLeaderboard) GetAll() []models.Bet {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	// return a copy to avoid race
	result := make([]models.Bet, len(lb.data))
	copy(result, lb.data)
	return result
}

// GetLeaderboard API handler for clients
func GetLeaderboard(_ map[string]interface{}) (models.HandlerOK, models.HandlerError) {
	var (
		errR models.HandlerError
		resR models.HandlerOK
	)

	events.Emit("all", "leaderboard", Leaderboard.GetAll())

	// Success -
	resR.Type = "getLeaderboard"
	resR.Data = Leaderboard.GetAll()
	return resR, errR
}
