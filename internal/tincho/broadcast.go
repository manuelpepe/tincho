package tincho

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/manuelpepe/tincho/internal/game"
)

func (r *Room) BroadcastUpdate(update Update) {
	for _, player := range r.state.GetPlayers() {
		conn, ok := r.GetPlayer(player.ID)
		if !ok {
			// TODO: probably should stop everything as this shouldn-t happen
			continue
		}
		conn.SendUpdateOrDrop(update)
	}
}

func (r *Room) BroadcastUpdateExcept(update Update, player game.PlayerID) {
	for _, p := range r.state.GetPlayers() {
		if p.ID != player {
			conn, ok := r.GetPlayer(p.ID)
			if !ok {
				// TODO: probably should stop everything as this shouldn-t happen
				continue
			}
			conn.SendUpdateOrDrop(update)
		}
	}
}

func (r *Room) TargetedUpdate(player game.PlayerID, update Update) {
	for _, p := range r.state.GetPlayers() {
		if p.ID == player {
			conn, ok := r.GetPlayer(p.ID)
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

func (r *Room) broadcastGameConfig(cardInDeck int) error {
	data, err := json.Marshal(UpdateGameConfig{
		CardsInDeck: cardInDeck,
	})
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}
	r.BroadcastUpdate(Update{
		Type: UpdateTypeGameConfig,
		Data: json.RawMessage(data),
	})
	return nil
}

func (r *Room) sendRejoinState(player *Connection) error {
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
	data, err := json.Marshal(UpdateTypeRejoinData{
		Players:       r.state.GetPlayers(),
		CurrentTurn:   curTurn,
		CardInHand:    r.state.GetPendingStorage() != game.Card{},
		CardInHandVal: cardInHandVal,
		LastDiscarded: lastDiscarded,
	})
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}
	r.TargetedUpdate(player.ID, Update{
		Type: UpdateTypeRejoin,
		Data: data,
	})
	return nil
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

func (r *Room) broadcastStartGame(topDiscard game.Card) error {
	data, err := json.Marshal(UpdateStartNextRoundData{
		Players:    r.state.GetPlayers(),
		TopDiscard: topDiscard,
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

func (r *Room) broadcastPlayerFirstPeeked(playerID game.PlayerID, cards []game.Card) error {
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

func (r *Room) broadcastDraw(playerID game.PlayerID, source game.DrawSource, card game.Card) error {
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

func (r *Room) broadcastDiscard(playerID game.PlayerID, positions []int, discarded []game.Card, cycledPiles game.CycledPiles) error {
	updateData, err := json.Marshal(UpdateDiscardData{
		Player:         playerID,
		CardsPositions: positions,
		Cards:          discarded,
		CycledPiles:    cycledPiles,
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

func (r *Room) broadcastFailedDoubleDiscard(playerID game.PlayerID, positions []int, cards []game.Card, topOfDiscard game.Card, cycledPiles game.CycledPiles) error {
	updateData, err := json.Marshal(UpdateTypeFailedDoubleDiscardData{
		Player:         playerID,
		CardsPositions: positions,
		Cards:          cards,
		TopOfDiscard:   topOfDiscard,
		CycledPiles:    cycledPiles,
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

func (r *Room) broadcastCut(playerID game.PlayerID, withCount bool, declared int) error {
	players := r.state.GetPlayers()
	hands := make([][]game.Card, len(players))
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

func (r *Room) broadcastNextRound(topDiscard game.Card) error {
	data, err := json.Marshal(UpdateStartNextRoundData{
		Players:    r.state.GetPlayers(),
		TopDiscard: topDiscard,
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

func (r *Room) broadcastEndGame(scores []game.Round) error {
	data, err := json.Marshal(UpdateEndGameData{
		Rounds: scores,
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

func (r *Room) broadcastSwapCards(
	playerID game.PlayerID,
	positions []int,
	players []game.PlayerID,
	discarded game.Card,
	cycledPiles game.CycledPiles,
) error {
	updateData, err := json.Marshal(
		UpdateSwapCardsData{
			CardsPositions: positions,
			Players:        players,
		},
	)
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}
	r.BroadcastUpdate(Update{
		Type: UpdateTypeSwapCards,
		Data: json.RawMessage(updateData),
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
	updateData, err := json.Marshal(UpdatePeekCardData{
		CardPosition: cardIndex,
		Card:         peeked,
		Player:       peekedPlayer,
	})
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}
	r.TargetedUpdate(targetPlayer, Update{
		Type: UpdateTypePeekCard,
		Data: json.RawMessage(updateData),
	})

	// update without value for other players
	updateData, err = json.Marshal(UpdatePeekCardData{
		CardPosition: cardIndex,
		Player:       peekedPlayer,
	})
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}
	r.BroadcastUpdateExcept(Update{
		Type: UpdateTypePeekCard,
		Data: json.RawMessage(updateData),
	}, targetPlayer)

	if err := r.broadcastDiscard(targetPlayer, []int{-1}, []game.Card{discarded}, cycledPiles); err != nil {
		return fmt.Errorf("broadcastDiscard: %w", err)
	}

	if err := r.broadcastPassTurn(); err != nil {
		return fmt.Errorf("PassTurn: %w", err)
	}

	return nil
}
