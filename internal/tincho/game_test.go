package tincho

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPlayersJoinRoom(t *testing.T) {
	room := NewRoom("test")
	player1 := Player{ID: "p1"}
	player2 := Player{ID: "p2"}
	assert.Equal(t, len(room.Players), 0)
	assert.NoError(t, room.AddPlayer(player1))
	assert.Equal(t, len(room.Players), 1)
	assert.NoError(t, room.AddPlayer(player2))
	assert.Equal(t, len(room.Players), 2)
	assert.ErrorIs(t, room.AddPlayer(player2), ErrPlayerAlreadyInRoom)
}
