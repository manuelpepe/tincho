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
	PlayerID PlayerID
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
	CardPosition int      `json:"cardPosition"`
	Player       PlayerID `json:"player"`
}

type ActionSwapCardsData struct {
	CardPositions []int      `json:"cardPositions"`
	Players       []PlayerID `json:"players"`
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

var ErrNotRoomLeader = errors.New("not room leader")

func (r *Room) doStartGame(action Action) error {
	if r.state.GetPlayers()[0].ID != action.PlayerID {
		return ErrNotRoomLeader
	}
	if err := r.state.StartGame(); err != nil {
		return fmt.Errorf("tsm.StartGame: %w", err)
	}
	if err := r.broadcastStartGame(); err != nil {
		return fmt.Errorf("broadcastStartGame: %w", err)
	}
	return nil
}

func (r *Room) doPeekTwo(action Action) error {
	peekedCards, err := r.state.GetFirstPeek(action.PlayerID)
	if err != nil {
		return fmt.Errorf("GetFirstPeek: %w", err)
	}
	if err := r.broadcastPlayerFirstPeeked(action.PlayerID, peekedCards); err != nil {
		return fmt.Errorf("broadcastPlayerPeeked: %w", err)
	}
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
	if err := r.broadcastDraw(action.PlayerID, data.Source, card); err != nil {
		return fmt.Errorf("broadcastDraw: %w", err)
	}
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
	if errors.Is(err, ErrDiscardingNonEqualCards) {
		positions := []int{data.CardPosition, *data.CardPosition2}
		if err := r.broadcastFailedDoubleDiscard(action.PlayerID, positions, disc); err != nil {
			return fmt.Errorf("broadcastFailedDoubleDiscard: %w", err)
		}
	} else {
		positions := []int{data.CardPosition}
		if data.CardPosition2 != nil {
			positions = append(positions, *data.CardPosition2)
		}
		if err := r.broadcastDiscard(action.PlayerID, positions, disc); err != nil {
			return fmt.Errorf("broadcastFailedDoubleDiscard: %w", err)
		}
	}
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
	scores, finished, err := r.state.Cut(data.WithCount, data.Declared)
	if err != nil {
		return err
	}
	if err := r.broadcastCut(action.PlayerID, data.WithCount, data.Declared); err != nil {
		return fmt.Errorf("broadcastCut: %w", err)
	}
	if finished {
		if err := r.broadcastEndGame(scores); err != nil {
			return fmt.Errorf("broadcastEndGame: %w", err)
		}
		r.Close()
	} else {
		if err := r.state.StartNextRound(); err != nil {
			return fmt.Errorf("StartNextRound: %w", err)
		}
		if err := r.broadcastNextRound(); err != nil {
			return fmt.Errorf("broadcastNextRound: %w", err)
		}
	}
	return nil
}

func (r *Room) doEffectPeekOwnCard(action Action) error {
	var data ActionPeekOwnCardData
	if err := json.Unmarshal(action.Data, &data); err != nil {
		return fmt.Errorf("json.Unmarshal: %w", err)
	}
	card, discarded, err := r.state.UseEffectPeekOwnCard(data.CardPosition)
	if err != nil {
		return err
	}
	if err := r.sendPeekToPlayer(action.PlayerID, action.PlayerID, data.CardPosition, card); err != nil {
		return fmt.Errorf("broadcastDiscard: %w", err)
	}
	updateData, err := json.Marshal(UpdateDiscardData{
		Player:         action.PlayerID,
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

func (r *Room) doEffectPeekCartaAjena(action Action) error {
	var data ActionPeekCartaAjenaData
	if err := json.Unmarshal(action.Data, &data); err != nil {
		return fmt.Errorf("json.Unmarshal: %w", err)
	}
	card, discarded, err := r.state.UseEffectPeekCartaAjena(data.Player, data.CardPosition)
	if err != nil {
		return err
	}
	if err := r.sendPeekToPlayer(action.PlayerID, data.Player, data.CardPosition, card); err != nil {
		return fmt.Errorf("broadcastDiscard: %w", err)
	}
	if err := r.broadcastDiscard(action.PlayerID, []int{-1}, []Card{discarded}); err != nil {
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
	discarded, err := r.state.UseEffectSwapCards(data.Players, data.CardPositions)
	if err != nil {
		return err
	}
	if err := r.broadcastSwapCards(action.PlayerID, data.CardPositions, data.Players, discarded); err != nil {
		return fmt.Errorf("broadcastSwapCards: %w", err)
	}
	return nil
}
