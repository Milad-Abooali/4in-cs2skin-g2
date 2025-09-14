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
var BetsByMultiplier map[float64][]models.Bet

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
	xp := int(profile["xp"].(float64))
	avatar := fmt.Sprintf("https://static.cs2skin.com/files/avatars/users/%s.webp", utils.MD5UserID(userID))
	displayName := fmt.Sprintf("%v", profile["display_name"])
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
		strconv.FormatInt(LiveGame.ID, 10),
		utils.RoundToTwoDigits(bet),
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
		ID:          0,
		Bet:         utils.RoundToTwoDigits(bet),
		GameID:      LiveGame.ID,
		UserID:      int64(userID),
		Avatar:      avatar,
		XP:          xp,
		DisplayName: displayName,
		Multiplier:  utils.RoundToTwoDigits(multiplier),
		CreatedAt:   time.Now().UTC(),
	}

	// Insert to Database
	betJSON, err := json.Marshal(newBet)
	if err != nil {
		log.Fatalln("failed to marshal game:", err)
	}
	// Sanitize and build query
	query := fmt.Sprintf(
		`INSERT INTO g2_bets (user_id, game_id, bet) 
				VALUES (%d, %d, '%s')`,
		newBet.UserID,
		newBet.GameID,
		string(betJSON),
	)

	// gRPC Call Insert User
	res, err := grpcclient.SendQuery(query)
	if err != nil || res == nil || res.Status != "ok" {
		errR.Type = "DB_ERROR_GRPC"
		errR.Code = 8000
		return resR, errR
	}
	dataDB := res.Data.GetFields()
	newID := int64(dataDB["inserted_id"].GetNumberValue())
	if newID < 1 {
		errR.Type = "DB_ERROR_RES"
		errR.Code = 8000
		return resR, errR
	}

	// Update Game ID
	newBet.ID = newID

	// Update Live Bets
	LiveBets[int64(userID)] = append(LiveBets[int64(userID)], newBet)
	events.Emit("all", "liveBets", LiveBets)

	BetsByMultiplier[newBet.Multiplier] = append(BetsByMultiplier[newBet.Multiplier], newBet)

	// Success
	resR.Type = "addBet"
	resR.Data = newBet
	return resR, errR
}

func CheckoutBet(data map[string]interface{}) (models.HandlerOK, models.HandlerError) {
	var (
		errR models.HandlerError
		resR models.HandlerOK
	)

	// Check Live Game
	if LiveGame.GameState != StateRunning {
		errR.Type = "GAME_NOT_RUNNING"
		errR.Code = 8002
		return resR, errR
	}

	multiplier := LiveGame.Multiplier

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

	// Check Bet
	betID, vErr, ok := validate.RequireInt(data, "betID")
	if !ok {
		return resR, vErr
	}

	// Get Bet
	log.Println(userID, betID)
	bet, ok := getBet(int64(userID), int(betID))
	if !ok {
		errR.Type = "BET_NOT_FOUND"
		errR.Code = 8003
		return resR, errR
	}

	if bet.Multiplier >= multiplier {
		errR.Type = "BET_ALREADY_CRASHED"
		errR.Code = 8004
		return resR, errR
	}

	if bet.Payout > 0 {
		errR.Type = "BET_ALREADY_PAID"
		errR.Code = 8005
		return resR, errR
	}

	// Win Price
	winAmount := utils.RoundToTwoDigits(bet.Bet * multiplier)

	// Add Transaction
	Transaction, err := utils.AddTransaction(
		userID,
		"game_win",
		"2",
		winAmount,
		strconv.FormatInt(bet.ID, 10),
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

	bet.Payout = winAmount
	bet.CheckoutBy = "User"

	// Update Live Bets
	LiveBets[int64(userID)][betID] = bet

	// Update DB
	betJSON, err := json.Marshal(bet)
	if err != nil {
		log.Fatalln("Failed to marshal bet:", err)
	}
	// Sanitize and build query
	query := fmt.Sprintf(
		`Update g2_bets SET bet = '%s', WHERE id = %d`,
		string(betJSON),
		bet.ID,
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

	events.Emit("all", "liveBets", LiveBets)

	// Success
	resR.Type = "checkOutBet"
	resR.Data = nil
	return resR, errR
}

func GetLiveBets(_ map[string]interface{}) (models.HandlerOK, models.HandlerError) {
	var (
		errR models.HandlerError
		resR models.HandlerOK
	)

	events.Emit("all", "liveBets", LiveBets)

	// Success
	resR.Type = "getLiveBets"
	return resR, errR
}

func processStep(multiplier float64) {
	bets, ok := BetsByMultiplier[multiplier]
	if !ok {
		return
	}
	for _, bet := range bets {
		go func(b models.Bet) {
			payout := utils.RoundToTwoDigits(b.Bet * multiplier)
			sendPayout(b.UserID, b.ID, payout)
		}(bet)
	}
	// delete(BetsByMultiplier, multiplier)
}

func sendPayout(userID int64, betID int64, payout float64) {

	// Get Bet
	bet, ok := getBet(userID, int(betID))
	if !ok {
		return
	}

	if bet.Payout > 0 {
		return
	}

	// Add Transaction
	Transaction, err := utils.AddTransaction(
		int(userID),
		"game_win",
		"2",
		payout,
		strconv.FormatInt(bet.ID, 10),
		"Crash",
	)
	if err != nil {
		return
	}
	_, status, _ := utils.SafeExtractErrorStatus(Transaction)
	if status != 1 {
		return
	}

	bet.Payout = payout
	bet.CheckoutBy = "Multiplier"

	// Update Live Bets
	LiveBets[userID][betID] = bet

	// Update DB
	betJSON, err := json.Marshal(bet)
	if err != nil {
		log.Fatalln("Failed to marshal bet:", err)
	}
	// Sanitize and build query
	query := fmt.Sprintf(
		`Update g2_bets SET bet = '%s', WHERE id = %d`,
		string(betJSON),
		bet.ID,
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

	events.Emit("all", "liveBets", LiveBets)

	// Success
	return
}

func getBet(userID int64, betID int) (models.Bet, bool) {
	bets, ok := LiveBets[userID]
	if !ok {
		return models.Bet{}, false
	}
	if betID < 0 || betID >= len(bets) {
		return models.Bet{}, false
	}
	return bets[betID], true
}
