package tincho

import (
	"encoding/json"
	"fmt"
)

type ActionType string

const (
	ActionStart          ActionType = "start"
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

type DrawSource string

const (
	DrawSourcePile    DrawSource = "pile"
	DrawSourceDiscard DrawSource = "discard"
)

type DrawAction struct {
	Source DrawSource `json:"source"`
}

type DiscardAction struct {
	// cardPosition = -1 means the card pending storage
	CardPosition int `json:"cardPosition"`
}

type CutAction struct {
	WithCount bool `json:"withCount"`
	Declared  int  `json:"declared"`
}

func (r *Room) PassTurn() {
	r.CurrentTurn = (r.CurrentTurn + 1) % len(r.Players)
}

func (r *Room) doStartGame(action Action) error {
	r.Playing = true
	r.BroadcastUpdate(Update{Type: UpdateTypeStart})
	return nil
}

func (r *Room) doDraw(action Action) error {
	var data DrawAction
	if err := json.Unmarshal(action.Data, &data); err != nil {
		return fmt.Errorf("json.Unmarshal: %w", err)
	}
	card, err := r.DrawCard(data.Source)
	if err != nil {
		return fmt.Errorf("DrawCard: %w", err)
	}
	mesageWithInfo, err := json.Marshal(UpdateDrawData{
		Source: data.Source,
		Card:   card,
	})
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}
	r.TargetedUpdate(action.PlayerID, Update{
		Type: UpdateTypeDraw,
		Data: json.RawMessage(mesageWithInfo),
	})
	messageNoInfo, err := json.Marshal(UpdateDrawData{
		Source: data.Source,
	})
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}
	r.BroadcastUpdate(Update{
		Type: UpdateTypeDraw,
		Data: json.RawMessage(messageNoInfo),
	})
	return nil
}

func (r *Room) DrawCard(source DrawSource) (Card, error) {
	if len(r.DrawPile) == 0 {
		if err := r.ReshufflePiles(); err != nil {
			return Card{}, fmt.Errorf("ReshufflePiles: %w", err)
		}
	}
	card, err := r.drawFromSource(source)
	if err != nil {
		return Card{}, fmt.Errorf("drawFromSource: %w", err)
	}
	r.PendingStorage = card
	return card, nil
}

func (r *Room) drawFromSource(source DrawSource) (Card, error) {
	switch source {
	case DrawSourcePile:
		return r.DrawPile.Draw()
	case DrawSourceDiscard:
		return r.DiscardPile.Draw()
	default:
		return Card{}, fmt.Errorf("invalid source: %s", source)
	}
}

func (r *Room) doDiscard(action Action) error {
	var data DiscardAction
	if err := json.Unmarshal(action.Data, &data); err != nil {
		return fmt.Errorf("json.Unmarshal: %w", err)
	}
	if err := r.DiscardCard(action.PlayerID, data.CardPosition); err != nil {
		return fmt.Errorf("DiscardCard: %w", err)
	}
	r.PassTurn()
	return nil
}

func (r *Room) DiscardCard(playerID string, card int) error {
	if card == -1 {
		r.DiscardPile = append(r.DiscardPile, r.PendingStorage)
		r.PendingStorage = Card{}
		return nil
	}
	player, exists := r.GetPlayer(playerID)
	if !exists {
		return fmt.Errorf("Unkown player: %s", playerID)
	}
	if card < -1 || card >= len(player.Hand) {
		return fmt.Errorf("invalid card position: %d", card)
	}
	// add card to top of start of discard pile
	r.DiscardPile = append([]Card{player.Hand[card]}, r.DiscardPile...)
	// remove card from hand and replace with stored card
	player.Hand[card] = r.PendingStorage
	r.PendingStorage = Card{}
	return nil
}

func (r *Room) doCut(action Action) error {
	var data CutAction
	if err := json.Unmarshal(action.Data, &data); err != nil {
		return fmt.Errorf("json.Unmarshal: %w", err)
	}
	r.Cut(data.WithCount, data.Declared)
	r.PassTurn()
	return nil
}

func (r *Room) Cut(withCount bool, decaled int) {}
