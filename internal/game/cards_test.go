package game

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCardRemove(t *testing.T) {
	ix := 1
	hand := Hand{Card{Suit: SuitClubs, Value: 1}, Card{Suit: SuitClubs, Value: 2}, Card{Suit: SuitClubs, Value: 3}}
	card := hand[ix]
	hand.Remove(ix)
	assert.Len(t, hand, 2)
	assert.NotContains(t, hand, card, "card not removed")
}
