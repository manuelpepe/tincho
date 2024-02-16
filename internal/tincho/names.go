package tincho

import (
	"fmt"
	"math/rand"
	"time"
)

var adjectives = []string{"Big", "Small", "Fast", "Slow", "Bright", "Dark", "Cold", "Hot", "Loud", "Quiet"}
var nouns = []string{"Dog", "Cat", "Car", "House", "Tree", "Mountain", "River", "Ocean", "Sun", "Moon"}

func RandomBotName() string {
	rand.Seed(time.Now().UnixNano())

	adjIndex := rand.Intn(len(adjectives))
	nounIndex := rand.Intn(len(nouns))

	number := rand.Intn(90) + 10

	adj := adjectives[adjIndex]
	noun := nouns[nounIndex]
	return fmt.Sprintf("[bot]%s%s%d", adj, noun, number)
}
