package provablyfair

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"math"
)

var (
	ServerSeed = "SERVER_SECRET"
	ClientSeed = "PLAYER_SEED"
)

func GenerateServerSeed() (string, string) {
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		panic(err)
	}
	seed := hex.EncodeToString(bytes)
	hash := sha256.Sum256([]byte(seed))
	return seed, hex.EncodeToString(hash[:]) // (serverSeed, serverSeedHash)
}

func CalculateCrashMultiplier(serverSeed string) float64 {
	// Step 1: Decode seed
	seedBytes, err := hex.DecodeString(serverSeed)
	if err != nil || len(seedBytes) == 0 {
		return 1.0 // fallback
	}

	// Step 2: HMAC with constant key (optional, for fairness)
	h := hmac.New(sha256.New, []byte("CrashGame"))
	h.Write(seedBytes)
	hash := h.Sum(nil)

	// Step 3: Convert first 8 bytes to uint64
	num := binary.BigEndian.Uint64(hash[:8])

	// Step 4: Normalize to float between 0 and 1
	ratio := float64(num) / float64(math.MaxUint64)

	// Step 5: Apply crash formula
	if ratio < 0.01 {
		return 1.0 // instant crash
	}

	// Example formula: multiplier = floor(100 / (1 - ratio)) / 100
	multiplier := math.Floor(100/(1.0-ratio)) / 100.0

	// Clamp to max multiplier (e.g. 100x)
	if multiplier > 100.0 {
		multiplier = 100.0
	}

	return multiplier
}
