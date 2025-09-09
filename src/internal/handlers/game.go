package handlers

import (
	"encoding/json"
	"github.com/Milad-Abooali/4in-cs2skin-g2/src/internal/events"
	"github.com/Milad-Abooali/4in-cs2skin-g2/src/internal/grpcclient"
	"github.com/Milad-Abooali/4in-cs2skin-g2/src/internal/models"
	"github.com/Milad-Abooali/4in-cs2skin-g2/src/internal/provablyfair"
	"google.golang.org/protobuf/types/known/structpb"
	"log"
	"time"
)

var LiveGame *models.LiveGame

var LiveBets map[int64][]models.Bet

func FillLiveGame() (bool, models.HandlerError) {
	var (
		errR       models.HandlerError
		dbLiveGame *structpb.ListValue
	)

	log.Println("Fill Live Game...")

	// Sanitize and build query
	query := `SELECT game FROM g2_games WHERE is_live=1`

	// gRPC Call
	res, err := grpcclient.SendQuery(query)
	if err != nil || res == nil || res.Status != "ok" {
		errR.Type = "PROFILE_GRPC_ERROR"
		errR.Code = 1033
		if res != nil {
			errR.Data = res.Error
		}
		return false, errR
	}
	// Extract gRPC struct
	dataDB := res.Data.GetFields()
	// DB result rows count
	exist := dataDB["count"].GetNumberValue()
	if exist == 0 {
		errR.Type = "DB_DATA"
		errR.Code = 1070
		return false, errR
	}
	// DB result rows get fields
	dbLiveGame = dataDB["rows"].GetListValue()

	for idx, row := range dbLiveGame.Values {
		structRow := row.GetStructValue()
		battleJSON := structRow.Fields["game"].GetStringValue() // JSON string

		var b models.LiveGame
		err := json.Unmarshal([]byte(battleJSON), &b)
		if err != nil {
			log.Println("Failed to unmarshal game:", err)
			continue
		}

		key := int64(b.ID)
		if key == 0 {
			key = int64(idx + 1)
		}

		LiveGame = &b
	}

	if LiveGame == nil {
		NextGame(0)
	} else {
		// Run Game
	}
	return true, errR
}

func NextGame(id int64) models.Game {
	seed, seedHash := provablyfair.GenerateServerSeed()
	newGame := models.Game{
		ID:             id,
		StartAt:        time.Now().UTC(),
		EndAt:          nil,
		Multiplier:     0.00,
		ServerSeedHash: seedHash,
		ServerSeed:     seed,
	}

	// Save to Database

	//gRPC
	newGame.ID = id

	// Add To Live Game
	LiveGame = &models.LiveGame{
		ID:             newGame.ID,
		GameState:      1,
		ServerSeedHash: newGame.ServerSeed,
		Multiplier:     newGame.Multiplier,
		ServerTime:     time.Now().UnixMilli(),
	}
	events.Emit("all", "liveGame", LiveGame)
	return newGame
}
