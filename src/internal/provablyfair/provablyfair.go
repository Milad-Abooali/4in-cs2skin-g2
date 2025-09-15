package provablyfair

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"math"
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

type RangeWeight struct {
	Min    float64
	Max    float64
	Weight float64
}

var weightedRanges = []RangeWeight{
	{Min: 0.01, Max: 1, Weight: 0},
	{Min: 1.01, Max: 5, Weight: 0.90},
	{Min: 5.01, Max: 10, Weight: 0.10},
}

func CalculateCrashMultiplier(serverSeed string) float64 {
	seedBytes, err := hex.DecodeString(serverSeed)
	if err != nil || len(seedBytes) == 0 {
		return 1.0
	}
	h := hmac.New(sha256.New, []byte("CrashGame"))
	h.Write(seedBytes)
	hash := h.Sum(nil)
	num := binary.BigEndian.Uint64(hash[:8])
	ratio := float64(num) / float64(math.MaxUint64)
	accum := 0.0
	var selected RangeWeight
	for _, r := range weightedRanges {
		accum += r.Weight
		if ratio <= accum {
			selected = r
			break
		}
	}
	num2 := binary.BigEndian.Uint64(hash[8:16])
	ratio2 := float64(num2) / float64(math.MaxUint64)
	multiplier := selected.Min + ratio2*(selected.Max-selected.Min)
	return math.Round(multiplier*100) / 100
}
