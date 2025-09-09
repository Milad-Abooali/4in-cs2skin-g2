package handlers

import (
	"encoding/json"
	"github.com/Milad-Abooali/4in-cs2skin-g2/src/internal/events"
	"github.com/Milad-Abooali/4in-cs2skin-g2/src/internal/grpcclient"
	"github.com/Milad-Abooali/4in-cs2skin-g2/src/internal/models"
	"github.com/Milad-Abooali/4in-cs2skin-g2/src/internal/provablyfair"
	"github.com/Milad-Abooali/4in-cs2skin-g2/src/utils"
	"google.golang.org/protobuf/types/known/structpb"
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

	// Update Game ID
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
	time.Sleep(5000 * time.Millisecond)

	// Waiting For Bets

	// Force Start
	log.Println(newGame)
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
			time.Sleep(100 * time.Millisecond)
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
	game.
		// Emit History

		// Call Next Game
		NextGame(game.ID + 1)
}
