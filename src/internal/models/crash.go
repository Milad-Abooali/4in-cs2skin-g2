package models

import "time"

type Heartbeat struct {
	GameId     float64   `json:"gameId"`
	Multiplier float64   `json:"multiplier"`
	GameState  int       `json:"gameState"`
	ServerTime time.Time `json:"serverTime"`
}
