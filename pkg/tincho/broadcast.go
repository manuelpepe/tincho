package tincho

import (
	"fmt"

	"github.com/manuelpepe/tincho/pkg/game"
)

func (r *Room) BroadcastUpdate(update Typed) {
	for _, player := range r.state.GetPlayers() {
		conn, ok := r.getPlayer(player.ID)
		if !ok {
			// TODO: probably should stop everything as this shouldn-t happen
			continue
		}
		conn.SendUpdateOrDrop(update)
	}
}

func (r *Room) BroadcastUpdateExcept(update Typed, player game.PlayerID) {
	for _, p := range r.state.GetPlayers() {
		if p.ID != player {
			conn, ok := r.getPlayer(p.ID)
			if !ok {
				// TODO: probably should stop everything as this shouldn-t happen
				continue
			}
			conn.SendUpdateOrDrop(update)
		}
	}
}

func (r *Room) TargetedUpdate(player game.PlayerID, update Typed) {
	for _, p := range r.state.GetPlayers() {
		if p.ID == player {
			conn, ok := r.getPlayer(p.ID)
			if !ok {
				// TODO: probably should stop everything as this shouldn-t happen
				continue
			}
			conn.SendUpdateOrDrop(update)
			return
		}
	}
}

func (r *Room) TargetedError(player game.PlayerID, err error) {
	r.TargetedUpdate(player, Update[UpdateErrorData]{
		Type: UpdateTypeError,
		Data: UpdateErrorData{
			Message: err.Error(),
		},
	})
}

func (r *Room) broadcastGameConfig(cardInDeck int) error {
	r.BroadcastUpdate(Update[UpdateGameConfig]{
		Type: UpdateTypeGameConfig,
		Data: UpdateGameConfig{
			CardsInDeck: cardInDeck,
		},
	})
	return nil
}

func (r *Room) sendRejoinState(player *Connection, cardsInDeck int, cardsInDrawPile int) error {
	curTurn := r.state.PlayerToPlay().ID
	pendStorage := r.state.GetPendingStorage()
	var cardInHandVal *game.Card
	if (player.ID == curTurn && pendStorage != game.Card{}) {
		cardInHandVal = &pendStorage
	}
	var lastDiscarded *game.Card
	if r.state.CountDiscardPile() > 0 {
		v := r.state.LastDiscarded()
		lastDiscarded = &v
	}
	r.TargetedUpdate(player.ID, Update[UpdateTypeRejoinData]{
		Type: UpdateTypeRejoin,
		Data: UpdateTypeRejoinData{
			Players:         r.getMarshalledPlayers(),
			CurrentTurn:     curTurn,
			CardInHand:      r.state.GetPendingStorage() != game.Card{},
			CardInHandVal:   cardInHandVal,
			LastDiscarded:   lastDiscarded,
			CardsInDeck:     cardsInDeck,
			CardsInDrawPile: cardsInDrawPile,
		},
	})
	return nil
}

func (r *Room) broadcastPassTurn() error {
	r.BroadcastUpdate(Update[UpdateTurnData]{
		Type: UpdateTypeTurn,
		Data: UpdateTurnData{
			Player: r.state.PlayerToPlay().ID,
		},
	})
	return nil
}

func (r *Room) broadcastStartGame(topDiscard game.Card) error {
	r.BroadcastUpdate(Update[UpdateStartNextRoundData]{
		Type: UpdateTypeGameStart,
		Data: UpdateStartNextRoundData{
			Players:    r.getMarshalledPlayers(),
			TopDiscard: topDiscard,
		},
	})
	return nil
}

func (r *Room) broadcastPlayerFirstPeeked(playerID game.PlayerID, cards []game.Card) error {
	// broadcast UpdateTypePlayerPeeked without cards
	r.BroadcastUpdateExcept(Update[UpdatePlayerFirstPeekedData]{
		Type: UpdateTypePlayerFirstPeeked,
		Data: UpdatePlayerFirstPeekedData{
			Player: playerID,
		},
	}, playerID)

	// target UpdateTypePlayerPeeked with cards to player
	r.TargetedUpdate(playerID, Update[UpdatePlayerFirstPeekedData]{
		Type: UpdateTypePlayerFirstPeeked,
		Data: UpdatePlayerFirstPeekedData{
			Player: playerID,
			Cards:  cards,
		},
	})
	return nil
}

func (r *Room) broadcastDraw(playerID game.PlayerID, source game.DrawSource, card game.Card) error {
	// target UpdateTypeDraw with card
	r.TargetedUpdate(playerID, Update[UpdateDrawData]{
		Type: UpdateTypeDraw,
		Data: UpdateDrawData{
			Player: playerID,
			Source: source,
			Card:   card,
			Effect: card.GetEffect(),
		},
	})

	// broadcast UpdateTypeDraw without card
	r.BroadcastUpdateExcept(Update[UpdateDrawData]{
		Type: UpdateTypeDraw,
		Data: UpdateDrawData{
			Player: playerID,
			Source: source,
		},
	}, playerID)
	return nil
}

func (r *Room) broadcastDiscard(playerID game.PlayerID, positions []int, discarded []game.Card, cycledPiles game.CycledPiles) error {
	r.BroadcastUpdate(Update[UpdateDiscardData]{
		Type: UpdateTypeDiscard,
		Data: UpdateDiscardData{
			Player:         playerID,
			CardsPositions: positions,
			Cards:          discarded,
			CycledPiles:    cycledPiles,
		},
	})
	return nil
}

func (r *Room) broadcastFailedDoubleDiscard(playerID game.PlayerID, positions []int, cards []game.Card, topOfDiscard game.Card, cycledPiles game.CycledPiles) error {
	r.BroadcastUpdate(Update[UpdateTypeFailedDoubleDiscardData]{
		Type: UpdateTypeFailedDoubleDiscard,
		Data: UpdateTypeFailedDoubleDiscardData{
			Player:         playerID,
			CardsPositions: positions,
			Cards:          cards,
			TopOfDiscard:   topOfDiscard,
			CycledPiles:    cycledPiles,
		},
	})
	return nil
}

func (r *Room) broadcastCut(playerID game.PlayerID, withCount bool, declared int) error {
	players := r.state.GetPlayers()
	hands := make([][]game.Card, len(players))
	for ix := range players {
		hands[ix] = players[ix].Hand
	}
	marshalled := make([]MarshalledPlayer, 0, len(players))
	for _, p := range players {
		marshalled = append(marshalled, NewMarshalledPlayer(p))
	}
	r.BroadcastUpdate(Update[UpdateCutData]{
		Type: UpdateTypeCut,
		Data: UpdateCutData{
			Player:    playerID,
			WithCount: withCount,
			Declared:  declared,
			Players:   marshalled,
			Hands:     hands,
		},
	})
	return nil
}

func (r *Room) broadcastNextRound(topDiscard game.Card) error {
	r.BroadcastUpdate(Update[UpdateStartNextRoundData]{
		Type: UpdateTypeStartNextRound,
		Data: UpdateStartNextRoundData{
			Players:    r.getMarshalledPlayers(),
			TopDiscard: topDiscard,
		},
	})
	return nil
}

func (r *Room) broadcastEndGame(scores []game.Round) error {
	r.BroadcastUpdate(Update[UpdateEndGameData]{
		Type: UpdateTypeEndGame,
		Data: UpdateEndGameData{
			Rounds: scores,
		},
	})
	return nil
}

func (r *Room) broadcastSwapCards(
	playerID game.PlayerID,
	positions []int,
	players []game.PlayerID,
	discarded game.Card,
	cycledPiles game.CycledPiles,
) error {
	r.BroadcastUpdate(Update[UpdateSwapCardsData]{
		Type: UpdateTypeSwapCards,
		Data: UpdateSwapCardsData{
			CardsPositions: positions,
			Players:        players,
		},
	})

	if err := r.broadcastDiscard(playerID, []int{-1}, []game.Card{discarded}, cycledPiles); err != nil {
		return fmt.Errorf("broadcastDiscard: %w", err)
	}

	if err := r.broadcastPassTurn(); err != nil {
		return fmt.Errorf("PassTurn: %w", err)
	}
	return nil
}

func (r *Room) broadcastPeek(
	targetPlayer game.PlayerID,
	peekedPlayer game.PlayerID,
	cardIndex int,
	peeked game.Card,
	discarded game.Card,
	cycledPiles game.CycledPiles,
) error {
	// update with value for player peeking
	r.TargetedUpdate(targetPlayer, Update[UpdatePeekCardData]{
		Type: UpdateTypePeekCard,
		Data: UpdatePeekCardData{
			CardPosition: cardIndex,
			Card:         peeked,
			Player:       peekedPlayer,
		},
	})

	// update without value for other players
	r.BroadcastUpdateExcept(Update[UpdatePeekCardData]{
		Type: UpdateTypePeekCard,
		Data: UpdatePeekCardData{
			CardPosition: cardIndex,
			Player:       peekedPlayer,
		},
	}, targetPlayer)

	if err := r.broadcastDiscard(targetPlayer, []int{-1}, []game.Card{discarded}, cycledPiles); err != nil {
		return fmt.Errorf("broadcastDiscard: %w", err)
	}

	if err := r.broadcastPassTurn(); err != nil {
		return fmt.Errorf("PassTurn: %w", err)
	}

	return nil
}
