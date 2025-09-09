package models

import "time"

type LiveGame struct {
	ID             int64   `json:"id"`
	Multiplier     float64 `json:"multiplier"`
	GameState      int     `json:"gameState"`
	ServerSeedHash string  `json:"serverSeedHash"`
	ServerTime     int64   `json:"serverTime"`
}

type CheckoutBy string

const (
	Crash      CheckoutBy = "Crash"
	User       CheckoutBy = "User"
	Multiplier CheckoutBy = "Multiplier"
)

const (
	StateWaiting  = 0
	StateRunning  = 1
	StateCrashed  = 2
	StateFinished = 3
)

type Bet struct {
	ID         int64       `json:"id"`
	UserID     int64       `json:"userID"`
	GameID     int64       `json:"gameID"`
	Bet        float64     `json:"bet"`
	Payout     float64     `json:"payout"`
	Multiplier float64     `json:"multiplier"`
	CheckoutOn float64     `json:"checkoutOn"`
	CheckoutBy *CheckoutBy `json:"CheckoutBy"`
	CreatedAt  time.Time   `json:"createdAt"`
}

type Game struct {
	ID             int64      `json:"id"`
	StartAt        time.Time  `json:"StartAt"`
	EndAt          *time.Time `json:"EndAt"`
	Multiplier     float64    `json:"Multiplier"`
	ServerSeedHash string     `json:"serverSeedHash"`
	ServerSeed     string     `json:"serverSeed"`
}
