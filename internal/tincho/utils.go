package tincho

import (
	"math/rand"
	"time"
)

// Function to generate a random string with a given length
func generateRandomString(length int) string {
	chars := "abcdefghijklmnopqrstuvwxyz"
	rand.Seed(time.Now().UnixNano())
	result := make([]byte, length)
	for i := range result {
		result[i] = chars[rand.Intn(len(chars))]
	}
	return string(result)
}
