package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/Milad-Abooali/4in-cs2skin-g2/src/internal/events"
	"github.com/Milad-Abooali/4in-cs2skin-g2/src/internal/grpcclient"
	"github.com/Milad-Abooali/4in-cs2skin-g2/src/internal/models"
	"github.com/Milad-Abooali/4in-cs2skin-g2/src/internal/validate"
	"github.com/Milad-Abooali/4in-cs2skin-g2/src/utils"
	"log"
	"strconv"
	"time"
)

var LiveBets map[int64][]models.Bet

func AddBet(data map[string]interface{}) (models.HandlerOK, models.HandlerError) {
	var (
		errR models.HandlerError
		resR models.HandlerOK
	)

	// Check Live Game
	if LiveGame.GameState != StateWaiting {
		errR.Type = "GAME_STARTED"
		errR.Code = 8001
		return resR, errR
	}

	// Check Token
	userJWT, vErr, ok := validate.RequireString(data, "token", false)
	if !ok {
		return resR, vErr
	}
	resp, err := utils.VerifyJWT(userJWT)
	if err != nil {
		return resR, models.HandlerError{}
	}
	errCode, status, errType := utils.SafeExtractErrorStatus(resp)
	if status != 1 {
		errR.Type = errType
		errR.Code = errCode
		if resp["data"] != nil {
			errR.Data = resp["data"]
		}
		return resR, errR
	}
	userData := resp["data"].(map[string]interface{})
	profile := userData["profile"].(map[string]interface{})
	userID := int(profile["id"].(float64))
	balanceStr := fmt.Sprintf("%v", profile["balance"])
	balance, err := strconv.ParseFloat(balanceStr, 64)
	if err != nil {
		balance = 0
	}

	// Check Bet
	bet, vErr, ok := validate.RequireFloat(data, "bet")
	if !ok {
		return resR, vErr
	}

	// Check Bet
	multiplier, vErr, ok := validate.RequireFloat(data, "multiplier")
	if !ok {
		return resR, vErr
	}

	// Check Balance
	if balance < bet {
		errR.Type = "INSUFFICIENT_BALANCE"
		errR.Code = 7001
		errR.Data = map[string]interface{}{
			"cost":    bet,
			"balance": balance,
		}
		return resR, errR
	}

	// Add Transaction
	Transaction, err := utils.AddTransaction(
		userID,
		"game_loss",
		"2",
		bet,
		"",
		"Crash",
	)
	if err != nil {
		return resR, models.HandlerError{}
	}
	errCode, status, errType = utils.SafeExtractErrorStatus(Transaction)
	if status != 1 {
		errR.Type = errType
		errR.Code = errCode
		if resp["data"] != nil {
			errR.Data = resp["data"]
		}
		return resR, errR
	}

	// Creat Bet
	newBet := models.Bet{
		ID:         0,
		Bet:        bet,
		GameID:     LiveGame.ID,
		UserID:     int64(userID),
		Multiplier: multiplier,
		CreatedAt:  time.Now().UTC(),
	}

	// Insert to Database
	betJSON, err := json.Marshal(newBet)
	if err != nil {
		log.Fatalln("failed to marshal game:", err)
	}
	// Sanitize and build query
	query := fmt.Sprintf(
		`INSERT INTO g2_bets (user_id, game_id, bet) 
				VALUES (%d, $d, '%s')`,
		newBet.UserID,
		newBet.GameID,
		string(betJSON),
	)
	// gRPC Call Insert User
	res, err := grpcclient.SendQuery(query)
	if err != nil || res == nil || res.Status != "ok" {
		errR.Type = "DB_ERROR"
		errR.Code = 80000
		return resR, errR
	}
	dataDB := res.Data.GetFields()
	newID := int64(dataDB["inserted_id"].GetNumberValue())
	if newID < 1 {
		errR.Type = "DB_ERROR"
		errR.Code = 8000
		return resR, errR
	}

	// Update Game ID
	newBet.ID = newID

	// Update Live Bets
	LiveBets[int64(userID)] = append(LiveBets[int64(userID)], newBet)
	events.Emit("all", "liveBets", LiveBets)

	// Success
	resR.Type = "addBet"
	resR.Data = nil
	return resR, errR
}

func CheckoutBet(data map[string]interface{}) (models.HandlerOK, models.HandlerError) {
	var (
		errR models.HandlerError
		resR models.HandlerOK
	)

	// Success
	resR.Type = "ping"
	resR.Data = time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	return resR, errR
}

func CrashBet(data map[string]interface{}) (models.HandlerOK, models.HandlerError) {
	var (
		errR models.HandlerError
		resR models.HandlerOK
	)

	// Success
	resR.Type = "ping"
	resR.Data = time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	return resR, errR
}

func Payout(data map[string]interface{}) (models.HandlerOK, models.HandlerError) {
	var (
		errR models.HandlerError
		resR models.HandlerOK
	)

	// Success
	resR.Type = "ping"
	resR.Data = time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	return resR, errR
}

func GetMyBets() {}

func GetLiveBets() {}
