package models

import "time"

type LiveGame struct {
	ID             int64   `json:"id"`
	Multiplier     float64 `json:"multiplier"`
	GameState      int     `json:"gameState"`
	ServerSeedHash string  `json:"serverSeedHash"`
	ServerTime     int64   `json:"serverTime"`
}

type Bet struct {
	ID         int64     `json:"id"`
	UserID     int64     `json:"userID"`
	GameID     int64     `json:"gameID"`
	Bet        float64   `json:"bet"`
	Payout     float64   `json:"payout"`
	Multiplier float64   `json:"multiplier"`
	CheckoutOn float64   `json:"checkoutOn"`
	CheckoutBy string    `json:"CheckoutBy"`
	CreatedAt  time.Time `json:"createdAt"`
}

type Game struct {
	ID             int64     `json:"id"`
	StartAt        time.Time `json:"StartAt"`
	EndAt          time.Time `json:"EndAt"`
	Multiplier     float64   `json:"Multiplier"`
	CrashAt        float64   `json:"CrashAt"`
	ServerSeedHash string    `json:"serverSeedHash"`
	ServerSeed     string    `json:"serverSeed"`
}
