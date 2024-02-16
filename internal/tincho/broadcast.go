package tincho

import (
	"encoding/json"
	"fmt"
	"log"
)

func (r *Room) BroadcastUpdate(update Update) {
	for _, player := range r.state.GetPlayers() {
		player.Updates <- update
	}
}

func (r *Room) BroadcastUpdateExcept(update Update, player string) {
	for _, p := range r.state.GetPlayers() {
		if p.ID != player {
			p.Updates <- update
		}
	}
}

func (r *Room) TargetedUpdate(player string, update Update) {
	for _, p := range r.state.GetPlayers() {
		if p.ID == player {
			p.Updates <- update
			return
		}
	}
}

func (r *Room) TargetedError(player string, err error) {
	data, err := json.Marshal(UpdateErrorData{
		Message: err.Error(),
	})
	if err != nil {
		log.Println(err)
		return
	}
	r.TargetedUpdate(player, Update{
		Type: UpdateTypeError,
		Data: data,
	})
}

func (r *Room) broadcastPassTurn() error {
	data, err := json.Marshal(UpdateTurnData{
		Player: r.state.PlayerToPlay().ID,
	})
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}
	r.BroadcastUpdate(Update{
		Type: UpdateTypeTurn,
		Data: json.RawMessage(data),
	})
	return nil
}

func (r *Room) broadcastStartGame() error {
	data, err := json.Marshal(UpdateStartNextRoundData{
		Players: r.state.GetPlayers(),
	})
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}
	r.BroadcastUpdate(Update{
		Type: UpdateTypeGameStart,
		Data: json.RawMessage(data),
	})
	return nil
}

func (r *Room) broadcastPlayerFirstPeeked(playerID string, cards []Card) error {
	// broadcast UpdateTypePlayerPeeked without cards
	data, err := json.Marshal(UpdatePlayerFirstPeekedData{
		Player: playerID,
	})
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}
	r.BroadcastUpdateExcept(Update{
		Type: UpdateTypePlayerFirstPeeked,
		Data: json.RawMessage(data),
	}, playerID)

	// target UpdateTypePlayerPeeked with cards to player
	data, err = json.Marshal(UpdatePlayerFirstPeekedData{
		Player: playerID,
		Cards:  cards,
	})
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}
	r.TargetedUpdate(playerID, Update{
		Type: UpdateTypePlayerFirstPeeked,
		Data: json.RawMessage(data),
	})
	return nil
}

func (r *Room) broadcastDraw(playerID string, source DrawSource, card Card) error {
	// target UpdateTypeDraw with card
	mesageWithInfo, err := json.Marshal(UpdateDrawData{
		Player: playerID,
		Source: source,
		Card:   card,
		Effect: card.GetEffect(),
	})
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}
	r.TargetedUpdate(playerID, Update{
		Type: UpdateTypeDraw,
		Data: json.RawMessage(mesageWithInfo),
	})

	// broadcast UpdateTypeDraw without card
	messageNoInfo, err := json.Marshal(UpdateDrawData{
		Player: playerID,
		Source: source,
	})
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}
	r.BroadcastUpdateExcept(Update{
		Type: UpdateTypeDraw,
		Data: json.RawMessage(messageNoInfo),
	}, playerID)
	return nil
}

func (r *Room) broadcastDiscard(playerID string, positions []int, discarded []Card) error {
	updateData, err := json.Marshal(UpdateDiscardData{
		Player:         playerID,
		CardsPositions: positions,
		Cards:          discarded,
	})
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}
	r.BroadcastUpdate(Update{
		Type: UpdateTypeDiscard,
		Data: json.RawMessage(updateData),
	})
	return nil
}

func (r *Room) broadcastFailedDoubleDiscard(playerID string, positions []int, cards []Card) error {
	updateData, err := json.Marshal(UpdateTypeFailedDoubleDiscardData{
		Player:         playerID,
		CardsPositions: positions,
		Cards:          cards,
	})
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}
	r.BroadcastUpdate(Update{
		Type: UpdateTypeFailedDoubleDiscard,
		Data: json.RawMessage(updateData),
	})
	return nil
}

func (r *Room) broadcastCut(playerID string, withCount bool, declared int) error {
	players := r.state.GetPlayers()
	hands := make([][]Card, len(players))
	for ix := range players {
		hands[ix] = players[ix].Hand
	}
	updateData, err := json.Marshal(UpdateCutData{
		Player:    playerID,
		WithCount: withCount,
		Declared:  declared,
		Players:   players,
		Hands:     hands,
	})
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}
	r.BroadcastUpdate(Update{
		Type: UpdateTypeCut,
		Data: json.RawMessage(updateData),
	})
	return nil
}

func (r *Room) broadcastNextRound() error {
	data, err := json.Marshal(UpdateStartNextRoundData{
		Players: r.state.GetPlayers(),
	})
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}
	r.BroadcastUpdate(Update{
		Type: UpdateTypeStartNextRound,
		Data: json.RawMessage(data),
	})
	return nil
}

func (r *Room) broadcastEndGame(scores [][]PlayerScore) error {
	data, err := json.Marshal(UpdateEndGameData{
		Scores: scores,
	})
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}
	r.BroadcastUpdate(Update{
		Type: UpdateTypeEndGame,
		Data: json.RawMessage(data),
	})
	return nil
}

func (r *Room) broadcastSwapCards(playerID string, positions []int, players []string, discarded Card) error {
	updateData, err := json.Marshal(
		UpdateSwapCardsData{
			CardPositions: positions,
			Players:       players,
		},
	)
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}
	r.BroadcastUpdate(Update{
		Type: UpdateTypeSwapCards,
		Data: json.RawMessage(updateData),
	})
	updateData, err = json.Marshal(UpdateDiscardData{
		Player:         playerID,
		CardsPositions: []int{-1},
		Cards:          []Card{discarded},
	})
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}
	r.BroadcastUpdate(Update{
		Type: UpdateTypeDiscard,
		Data: json.RawMessage(updateData),
	})
	if err := r.broadcastPassTurn(); err != nil {
		return fmt.Errorf("PassTurn: %w", err)
	}
	return nil
}

func (r *Room) sendPeekToPlayer(targetPlayer string, peekedPlayer string, cardIndex int, card Card) error {
	updateData, err := json.Marshal(UpdatePeekCardData{
		CardPosition: cardIndex,
		Card:         card,
		Player:       peekedPlayer,
	})
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}
	r.TargetedUpdate(targetPlayer, Update{
		Type: UpdateTypePeekCard,
		Data: json.RawMessage(updateData),
	})
	return nil
}
