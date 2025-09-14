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
	Username   string    `json:"username"`
	Avatar     string    `json:"avatar"`
	XPLevel    int       `json:"xpLevel"`
	GameID     int64     `json:"gameID"`
	Bet        float64   `json:"bet"`
	Payout     float64   `json:"payout"`
	Multiplier float64   `json:"multiplier"`
	CheckoutOn float64   `json:"checkoutOn"`
	CheckoutBy string    `json:"checkoutBy"`
	CreatedAt  time.Time `json:"createdAt"`
}

type Game struct {
	ID             int64     `json:"id"`
	StartAt        time.Time `json:"startAt"`
	EndAt          time.Time `json:"endAt"`
	Multiplier     float64   `json:"multiplier"`
	CrashAt        float64   `json:"crashAt"`
	ServerSeedHash string    `json:"serverSeedHash"`
	ServerSeed     string    `json:"serverSeed"`
}
