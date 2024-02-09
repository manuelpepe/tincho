package tincho

import (
	"encoding/json"
	"errors"
	"fmt"
)

type ActionType string

const (
	ActionStart          ActionType = "start"
	ActionFirstPeek      ActionType = "first_peek"
	ActionDraw           ActionType = "draw"
	ActionPeekOwnCard    ActionType = "effect_peek_own"
	ActionPeekCartaAjena ActionType = "effect_peek_carta_ajena"
	ActionSwapCards      ActionType = "effect_swap_card"
	ActionDiscard        ActionType = "discard"
	ActionCut            ActionType = "cut"
)

type Action struct {
	Type     ActionType      `json:"type"`
	Data     json.RawMessage `json:"data"`
	PlayerID string
}

type ActionDrawData struct {
	Source DrawSource `json:"source"`
}

type DrawSource string

const (
	DrawSourcePile    DrawSource = "pile"
	DrawSourceDiscard DrawSource = "discard"
)

type ActionPeekOwnCardData struct {
	CardPosition int `json:"cardPosition"`
}

type ActionPeekCartaAjenaData struct {
	CardPosition int    `json:"cardPosition"`
	Player       string `json:"player"`
}

type ActionSwapCardsData struct {
	CardPositions []int    `json:"cardPositions"`
	Players       []string `json:"players"`
}

type ActionDiscardData struct {
	// cardPosition = -1 means the card pending storage
	CardPosition  int  `json:"cardPosition"`
	CardPosition2 *int `json:"cardPosition2"`
}

type ActionCutData struct {
	WithCount bool `json:"withCount"`
	Declared  int  `json:"declared"`
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

func (r *Room) doStartGame(action Action) error {
	if err := r.state.StartGame(); err != nil {
		return fmt.Errorf("tsm.StartGame: %w", err)
	}
	data, err := json.Marshal(UpdateGameStart{
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

func (r *Room) doPeekTwo(action Action) error {
	peekedCards, err := r.state.GetFirstPeek(action.PlayerID)
	if err != nil {
		return fmt.Errorf("GetFirstPeek: %w", err)
	}

	// broadcast UpdateTypePlayerPeeked without cards
	data, err := json.Marshal(UpdatePlayerPeekedData{
		Player: action.PlayerID,
	})
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}
	r.BroadcastUpdateExcept(Update{
		Type: UpdateTypePlayerPeeked,
		Data: json.RawMessage(data),
	}, action.PlayerID)

	// target UpdateTypePlayerPeeked with cards to player
	data, err = json.Marshal(UpdatePlayerPeekedData{
		Player: action.PlayerID,
		Cards:  peekedCards,
	})
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}
	r.TargetedUpdate(action.PlayerID, Update{
		Type: UpdateTypePlayerPeeked,
		Data: json.RawMessage(data),
	})

	// if all players are ready, broadcast start
	if r.state.AllPlayersFirstPeeked() {
		if err := r.broadcastPassTurn(); err != nil {
			return fmt.Errorf("broadcastPassTurn: %w", err)
		}
	}
	return nil
}

func (r *Room) doDraw(action Action) error {
	var data ActionDrawData
	if err := json.Unmarshal(action.Data, &data); err != nil {
		return fmt.Errorf("json.Unmarshal: %w", err)
	}
	card, err := r.state.Draw(data.Source)
	if err != nil {
		return err
	}

	// target UpdateTypeDraw with card
	mesageWithInfo, err := json.Marshal(UpdateDrawData{
		Player: action.PlayerID,
		Source: data.Source,
		Card:   card,
		Effect: card.GetEffect(),
	})
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}
	r.TargetedUpdate(action.PlayerID, Update{
		Type: UpdateTypeDraw,
		Data: json.RawMessage(mesageWithInfo),
	})

	// broadcast UpdateTypeDraw without card
	messageNoInfo, err := json.Marshal(UpdateDrawData{
		Player: action.PlayerID,
		Source: data.Source,
	})
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}
	r.BroadcastUpdateExcept(Update{
		Type: UpdateTypeDraw,
		Data: json.RawMessage(messageNoInfo),
	}, action.PlayerID)
	return nil
}

func (r *Room) doDiscard(action Action) error {
	var data ActionDiscardData
	if err := json.Unmarshal(action.Data, &data); err != nil {
		return fmt.Errorf("json.Unmarshal: %w", err)
	}
	disc, err := r.state.Discard(data.CardPosition, data.CardPosition2)
	if err != nil && !errors.Is(err, ErrDiscardingNonEqualCards) {
		return err
	}
	var updateType UpdateType
	var updateData json.RawMessage
	if errors.Is(err, ErrDiscardingNonEqualCards) {
		updateType = UpdateTypeFailedDoubleDiscard
		updateData, err = json.Marshal(UpdateTypeFailedDoubleDiscardData{
			Player:         action.PlayerID,
			CardsPositions: []int{data.CardPosition, *data.CardPosition2}, // assumed cardpos2 not nil
			Cards:          disc,
		})
		if err != nil {
			return fmt.Errorf("json.Marshal: %w", err)
		}
	} else {
		updateType = UpdateTypeDiscard
		positions := []int{data.CardPosition}
		if data.CardPosition2 != nil {
			positions = append(positions, *data.CardPosition2)
		}
		updateData, err = json.Marshal(UpdateDiscardData{
			Player:         action.PlayerID,
			CardsPositions: positions,
			Cards:          disc,
		})
		if err != nil {
			return fmt.Errorf("json.Marshal: %w", err)
		}
	}
	r.BroadcastUpdate(Update{
		Type: updateType,
		Data: json.RawMessage(updateData),
	})
	if err := r.broadcastPassTurn(); err != nil {
		return fmt.Errorf("PassTurn: %w", err)
	}
	return nil
}

func (r *Room) doCut(action Action) error {
	var data ActionCutData
	if err := json.Unmarshal(action.Data, &data); err != nil {
		return fmt.Errorf("json.Unmarshal: %w", err)
	}
	if err := r.state.Cut(data.WithCount, data.Declared); err != nil {
		return err
	}
	// TODO: Add hands to broadcast
	updateData, err := json.Marshal(UpdateCutData{
		Player:    action.PlayerID,
		WithCount: data.WithCount,
		Declared:  data.Declared,
		Players:   r.state.GetPlayers(),
	})
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}
	r.BroadcastUpdate(Update{
		Type: UpdateTypeCut,
		Data: json.RawMessage(updateData),
	})
	// TODO: Reset deck, deal, reset turn, broadcast
	if r.state.IsWinConditionMet() {
		// TODO: Send winner
		r.BroadcastUpdate(Update{Type: UpdateTypeEndGame})
		r.Close()
	}
	return nil
}

func (r *Room) doEffectPeekOwnCard(action Action) error {
	var data ActionPeekOwnCardData
	if err := json.Unmarshal(action.Data, &data); err != nil {
		return fmt.Errorf("json.Unmarshal: %w", err)
	}
	card, err := r.state.UseEffectPeekOwnCard(data.CardPosition)
	if err != nil {
		return err
	}
	if err := r.sendPeekToPlayer(action.PlayerID, action.PlayerID, data.CardPosition, card); err != nil {
		return fmt.Errorf("broadcastDiscard: %w", err)
	}
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

func (r *Room) doEffectPeekCartaAjena(action Action) error {
	var data ActionPeekCartaAjenaData
	if err := json.Unmarshal(action.Data, &data); err != nil {
		return fmt.Errorf("json.Unmarshal: %w", err)
	}
	card, err := r.state.UseEffectPeekCartaAjena(data.CardPosition)
	if err != nil {
		return err
	}
	if err := r.sendPeekToPlayer(action.PlayerID, data.Player, data.CardPosition, card); err != nil {
		return fmt.Errorf("broadcastDiscard: %w", err)
	}
	if err := r.broadcastPassTurn(); err != nil {
		return fmt.Errorf("PassTurn: %w", err)
	}
	return nil
}

func (r *Room) doEffectSwapCards(action Action) error {
	var data ActionSwapCardsData
	if err := json.Unmarshal(action.Data, &data); err != nil {
		return fmt.Errorf("json.Unmarshal: %w", err)
	}
	if err := r.state.UseEffectSwapCards(data.Players, data.CardPositions); err != nil {
		return err
	}
	updateData, err := json.Marshal(UpdateSwapCardsData{
		CardPositions: data.CardPositions,
		Players:       data.Players,
	})
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}
	r.BroadcastUpdate(Update{
		Type: UpdateTypeSwapCards,
		Data: json.RawMessage(updateData),
	})
	if err := r.broadcastPassTurn(); err != nil {
		return fmt.Errorf("PassTurn: %w", err)
	}
	return nil
}
