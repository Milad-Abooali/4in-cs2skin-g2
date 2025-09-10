package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/Milad-Abooali/4in-cs2skin-g2/src/internal/events"
	"github.com/Milad-Abooali/4in-cs2skin-g2/src/internal/grpcclient"
	"github.com/Milad-Abooali/4in-cs2skin-g2/src/internal/models"
	"github.com/Milad-Abooali/4in-cs2skin-g2/src/internal/provablyfair"
	"github.com/Milad-Abooali/4in-cs2skin-g2/src/utils"
	"log"
	"time"
)

const (
	StateWaiting  = 0
	StateRunning  = 1
	StateCrashed  = 2
	StateFinished = 3
)

var LiveGame *models.LiveGame
var History = NewCrashHistory(50)

func NextGame(id int64) {
	serverSeed, serverSeedHash := provablyfair.GenerateServerSeed()
	crashAt := provablyfair.CalculateCrashMultiplier(serverSeed)
	newGame := models.Game{
		ID:             id,
		StartAt:        time.Now().UTC(),
		Multiplier:     0.00,
		CrashAt:        utils.RoundToTwoDigits(crashAt),
		ServerSeedHash: serverSeedHash,
		ServerSeed:     serverSeed,
	}

	// Insert to Database
	gameJSON, err := json.Marshal(newGame)
	if err != nil {
		log.Fatalln("failed to marshal game:", err)
	}
	// Sanitize and build query
	query := fmt.Sprintf(
		`INSERT INTO g2_games (server_seed,server_seed_hash, game) 
				VALUES ('%s', '%s', '%s')`,
		serverSeed,
		serverSeedHash,
		string(gameJSON),
	)
	// gRPC Call Insert User
	res, err := grpcclient.SendQuery(query)
	if err != nil || res == nil || res.Status != "ok" {
		log.Fatalln("DB_DATA:", err)
	}
	dataDB := res.Data.GetFields()
	newID := int64(dataDB["inserted_id"].GetNumberValue())
	if newID < 1 {
		log.Fatalln("DB_DATA:", err)
	}

	// Update Game ID
	newGame.ID = newID

	// Add To Live Game
	LiveGame = &models.LiveGame{
		ID:             newGame.ID,
		GameState:      StateWaiting,
		ServerSeedHash: newGame.ServerSeed,
		Multiplier:     newGame.Multiplier,
		ServerTime:     time.Now().UnixMilli(),
	}
	events.Emit("all", "liveGame", LiveGame)

	// Waiting For Bets
	time.Sleep(15000 * time.Millisecond)
	LiveGame.GameState = StateRunning

	// Force Start
	log.Println(newGame.ID, newGame.CrashAt)
	StartGameLoop(newGame)
}

func StartGameLoop(game models.Game) {
	go func() {
		multiplier := 0.01
		for {
			if LiveGame == nil || LiveGame.GameState != StateRunning {
				game.EndAt = time.Now().UTC()
				go EndGame(game)
				break
			}
			time.Sleep(250 * time.Millisecond)
			multiplier += 0.01
			LiveGame.Multiplier = utils.RoundToTwoDigits(multiplier)
			LiveGame.ServerTime = time.Now().UnixMilli()
			if LiveGame.Multiplier >= game.CrashAt {
				events.Emit("all", "crash", nil)
				LiveGame.GameState = StateCrashed
				LiveGame.Multiplier = game.CrashAt
			}
			events.Emit("all", "liveGame", LiveGame)
		}
	}()
}

func EndGame(game models.Game) {
	// Update DB
	gameJSON, err := json.Marshal(game)
	if err != nil {
		log.Fatalln("Failed to marshal game:", err)
	}
	// Sanitize and build query
	query := fmt.Sprintf(
		`Update g2_games SET game = '%s', is_live=0 WHERE id = %d`,
		string(gameJSON),
		game.ID,
	)
	res, err := grpcclient.SendQuery(query)
	if err != nil || res == nil || res.Status != "ok" {
		log.Fatalln("GRPC_ERROR", err)
	}
	dataDB := res.Data.GetFields()
	exist := dataDB["rows_affected"].GetNumberValue()
	if exist == 0 {
		log.Fatalln("NOT_UPDATED", dataDB)
	}
	LiveGame.GameState = StateFinished
	events.Emit("all", "liveGame", LiveGame)

	time.Sleep(1000 * time.Millisecond)

	// Emit History
	History.Add(game.CrashAt)
	events.Emit("all", "history", History.GetAll())

	// Call Next Game
	NextGame(game.ID + 1)
}
