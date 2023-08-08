package tincho

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPlayersJoinRoom(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	deck := NewDeck()
	room := NewRoomWithDeck(ctx, "test", deck)
	go room.Start()
	player1 := Player{ID: "p1"}
	player2 := Player{ID: "p2"}
	assert.Equal(t, len(room.Players), 0)
	assert.NoError(t, room.AddPlayer(player1))
	assert.Equal(t, len(room.Players), 1)
	assert.NoError(t, room.AddPlayer(player2))
	assert.Equal(t, len(room.Players), 2)
	assert.ErrorIs(t, room.AddPlayer(player2), ErrPlayerAlreadyInRoom)
}

func TestPlayerDrawsAndDiscards(t *testing.T) {
	// setup
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	deck := NewDeck()
	room := NewRoomWithDeck(ctx, "test", deck)
	go room.Start()
	player := Player{ID: "p1"}
	assert.NoError(t, room.AddPlayer(player))
	assert.NoError(t, room.Deal())
	startDrawLen := len(room.DrawPile)

	// draw
	expected := room.DrawPile[0]
	card, err := room.DrawCard(DrawSourcePile)
	assert.NoError(t, err, "draw card")
	assert.Equal(t, startDrawLen-1, len(room.DrawPile), "draw pile length")
	assert.Equal(t, expected, card, "pending storage")

	// discard drawn card
	assert.NoError(t, room.DiscardCard(player.ID, -1), "discard drawn card")
	assert.Equal(t, 1, len(room.DiscardPile), "one card discarded")
	assert.Equal(t, card, room.DiscardPile[0], "correct card discarded from draw")

	// draw again
	expected = room.DrawPile[0]
	new_card, err := room.DrawCard(DrawSourcePile)
	assert.NoError(t, err)
	assert.Equal(t, startDrawLen-2, len(room.DrawPile))
	assert.Equal(t, expected, new_card)
	assert.NotEqual(t, card, new_card)

	// store card and discard from hand
	toDiscard := room.Players[0].Hand[1]
	assert.NoError(t, room.DiscardCard(player.ID, 1), "discard second card from hand")
	assert.Equal(t, 2, len(room.DiscardPile), "two cards discarded")
	assert.Equal(t, toDiscard, room.DiscardPile[0], "correct card discarded from hand")
}
