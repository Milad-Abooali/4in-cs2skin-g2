package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/Milad-Abooali/4in-cs2skin-g2/src/apiapp"
	"github.com/Milad-Abooali/4in-cs2skin-g2/src/internal/events"
	"github.com/Milad-Abooali/4in-cs2skin-g2/src/internal/grpcclient"
	"github.com/Milad-Abooali/4in-cs2skin-g2/src/internal/models"
	"github.com/Milad-Abooali/4in-cs2skin-g2/src/internal/validate"
	"github.com/Milad-Abooali/4in-cs2skin-g2/src/utils"
	"log"
	"math"
	"strconv"
	"time"
)

var LiveBets map[int64][]models.Bet
var BetsByMultiplier = make(map[int][]models.Bet) // int key (multiplier*100)

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

	// Check Bet Limits per User
	userBets := LiveBets[int64(userID)]

	// User bets counts
	if len(userBets) >= 10 {
		errR.Type = "BET_LIMIT_REACHED"
		errR.Code = 8006
		return resR, errR
	}

	// User bets Amounts
	var userTotal float64
	for _, b := range userBets {
		userTotal += b.Bet
	}
	if userTotal+bet > 200 {
		errR.Type = "BET_LIMIT_REACHED"
		errR.Code = 8006
		return resR, errR
	}

	// Game Max Bet Amount
	var allTotal float64
	for _, bets := range LiveBets {
		for _, b := range bets {
			allTotal += b.Bet
		}
	}
	if allTotal+bet > 10000 {
		errR.Type = "GAME_MAX_BET_REACHED"
		errR.Code = 8007
		return resR, errR
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

	// Update BetsByMultiplier
	key := int(math.Round(newBet.Multiplier * 100)) // bucket with 2 decimals
	BetsByMultiplier[key] = append(BetsByMultiplier[key], newBet)

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
	// displayName := profile["display_name"].(string)

	// Check Bet
	betID, vErr, ok := validate.RequireInt(data, "betID")
	if !ok {
		return resR, vErr
	}

	// Get Bet
	bet, ok := getBet(int64(userID), betID)
	if !ok {
		errR.Type = "BET_NOT_FOUND"
		errR.Code = 8003
		return resR, errR
	}

	if bet.Multiplier <= multiplier {
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

	// Update DB
	betJSON, err := json.Marshal(bet)
	if err != nil {
		log.Fatalln("Failed to marshal bet:", err)
	}
	// Sanitize and build query
	query := fmt.Sprintf(
		`Update g2_bets SET bet = '%s' WHERE id = %d`,
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

	Leaderboard.Add(*bet)

	events.Emit("all", "liveBets", LiveBets)

	// Send Live Winner
	go sendLiveWinner(
		int64(userID),
		strconv.FormatFloat(bet.Bet, 'f', 2, 64),
		strconv.FormatFloat(bet.Multiplier, 'f', 2, 64),
		strconv.FormatFloat(bet.Payout, 'f', 2, 64),
	)

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
	for _, bets := range LiveBets {
		for _, bet := range bets {
			if bet.Payout == 0 && multiplier >= bet.Multiplier {
				payout := utils.RoundToTwoDigits(bet.Bet * multiplier)
				sendPayout(bet.UserID, bet.ID, payout)
			}
		}
	}
}

func sendPayout(userID int64, betID int64, payout float64) bool {
	// Get Bet by ID (returns pointer)
	bet, ok := getBet(userID, betID)
	if !ok {
		return false
	}

	if bet.Payout > 0 {
		return false
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
		return false
	}
	_, status, _ := utils.SafeExtractErrorStatus(Transaction)
	if status != 1 {
		return false
	}

	// Update bet in memory (no need to reassign slice element)
	bet.Payout = payout
	bet.CheckoutBy = "Multiplier"

	// Update DB
	betJSON, err := json.Marshal(bet)
	if err != nil {
		log.Fatalln("Failed to marshal bet:", err)
		return false
	}

	query := fmt.Sprintf(
		`UPDATE g2_bets SET bet = '%s' WHERE id = %d`,
		string(betJSON),
		bet.ID,
	)
	res, err := grpcclient.SendQuery(query)
	if err != nil || res == nil || res.Status != "ok" {
		log.Fatalln("GRPC_ERROR", err)
		return false
	}
	dataDB := res.Data.GetFields()
	exist := dataDB["rows_affected"].GetNumberValue()
	if exist == 0 {
		log.Fatalln("NOT_UPDATED", dataDB)
		return false
	}

	Leaderboard.Add(*bet)
	events.Emit("all", "liveBets", LiveBets)

	// Send Live Winner
	go sendLiveWinner(
		userID,
		strconv.FormatFloat(bet.Bet, 'f', 2, 64),
		strconv.FormatFloat(bet.Multiplier, 'f', 2, 64),
		strconv.FormatFloat(bet.Payout, 'f', 2, 64),
	)

	return true
}

func getBet(userID, betID int64) (*models.Bet, bool) {
	bets, ok := LiveBets[userID]
	if !ok {
		return nil, false
	}
	for i := range bets {
		if bets[i].ID == betID {
			return &bets[i], true
		}
	}
	return nil, false
}

func sendLiveWinner(userID int64, bet string, multiplier string, payout string) bool {
	apiAppErr := apiapp.InsertWinner(
		2,
		time.Now(),
		userID,
		bet,
		multiplier,
		payout,
	)
	if apiAppErr != nil {
		log.Println("Error:", apiAppErr)
		return false
	}
	return true
}
