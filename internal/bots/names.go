package bots

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/manuelpepe/tincho/internal/game"
)

var adjectives = []string{"Big", "Small", "Fast", "Slow", "Bright", "Dark", "Cold", "Hot", "Loud", "Quiet"}
var nouns = []string{"Dog", "Cat", "Car", "House", "Tree", "Mountain", "River", "Ocean", "Sun", "Moon"}

func RandomBotName() game.PlayerID {
	rand.Seed(time.Now().UnixNano())

	adjIndex := rand.Intn(len(adjectives))
	nounIndex := rand.Intn(len(nouns))

	number := rand.Intn(90) + 10

	adj := adjectives[adjIndex]
	noun := nouns[nounIndex]
	return game.PlayerID(fmt.Sprintf("[bot]%s%s%d", adj, noun, number))
}
