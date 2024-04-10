package tincho

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/manuelpepe/tincho/internal/game"
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
	PlayerID game.PlayerID
}

type Game_DrawSource string

type ActionDrawData struct {
	Source game.DrawSource `json:"source"`
}

type ActionPeekOwnCardData struct {
	CardPosition int `json:"cardPosition"`
}

type ActionPeekCartaAjenaData struct {
	CardPosition int           `json:"cardPosition"`
	Player       game.PlayerID `json:"player"`
}

type ActionSwapCardsData struct {
	CardPositions []int           `json:"cardPositions"`
	Players       []game.PlayerID `json:"players"`
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
	topDiscard, err := r.state.StartGame()
	if err != nil {
		return fmt.Errorf("tsm.StartGame: %w", err)
	}
	if err := r.broadcastStartGame(topDiscard); err != nil {
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

	var positions []int
	var cycledPiles game.CycledPiles
	var values []game.Card
	var err error

	if data.CardPosition2 == nil {
		var value game.Card
		value, cycledPiles, err = r.state.Discard(data.CardPosition)
		if err != nil {
			return err
		}
		positions = []int{data.CardPosition}
		values = []game.Card{value}
	} else {
		var disc []game.Card
		var topOfDiscardPile game.Card
		disc, topOfDiscardPile, cycledPiles, err = r.state.DiscardTwo(data.CardPosition, *data.CardPosition2)
		if err != nil && !errors.Is(err, game.ErrDiscardingNonEqualCards) {
			return err
		}

		if errors.Is(err, game.ErrDiscardingNonEqualCards) {
			positions := []int{data.CardPosition, *data.CardPosition2}
			err = r.broadcastFailedDoubleDiscard(action.PlayerID, positions, disc, topOfDiscardPile, cycledPiles)
			if err != nil {
				return fmt.Errorf("broadcastFailedDoubleDiscard: %w", err)
			}
			if err := r.broadcastPassTurn(); err != nil {
				return fmt.Errorf("PassTurn: %w", err)
			}
			return nil
		}

		positions = []int{data.CardPosition, *data.CardPosition2}
		values = disc
	}

	if err := r.broadcastDiscard(action.PlayerID, positions, values, cycledPiles); err != nil {
		return fmt.Errorf("broadcastDiscard: %w", err)
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
		// wait a few seconds before closing to ensure everyone recieves updates
		time.Sleep(10 * time.Second)
		r.Close()
	} else {
		topDiscard, err := r.state.StartNextRound()
		if err != nil {
			return fmt.Errorf("StartNextRound: %w", err)
		}
		if err := r.broadcastNextRound(topDiscard); err != nil {
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
	card, discarded, cycledPiles, err := r.state.UseEffectPeekOwnCard(data.CardPosition)
	if err != nil {
		return err
	}
	err = r.broadcastPeek(action.PlayerID, action.PlayerID, data.CardPosition, card, discarded, cycledPiles)
	if err != nil {
		return fmt.Errorf("broadcastDiscard: %w", err)
	}
	return nil
}

func (r *Room) doEffectPeekCartaAjena(action Action) error {
	var data ActionPeekCartaAjenaData
	if err := json.Unmarshal(action.Data, &data); err != nil {
		return fmt.Errorf("json.Unmarshal: %w", err)
	}
	card, discarded, cycledPiles, err := r.state.UseEffectPeekCartaAjena(data.Player, data.CardPosition)
	if err != nil {
		return err
	}
	err = r.broadcastPeek(action.PlayerID, data.Player, data.CardPosition, card, discarded, cycledPiles)
	if err != nil {
		return fmt.Errorf("broadcastDiscard: %w", err)
	}
	return nil
}

func (r *Room) doEffectSwapCards(action Action) error {
	var data ActionSwapCardsData
	if err := json.Unmarshal(action.Data, &data); err != nil {
		return fmt.Errorf("json.Unmarshal: %w", err)
	}
	discarded, cycledPiles, err := r.state.UseEffectSwapCards(data.Players, data.CardPositions)
	if err != nil {
		return err
	}
	err = r.broadcastSwapCards(action.PlayerID, data.CardPositions, data.Players, discarded, cycledPiles)
	if err != nil {
		return fmt.Errorf("broadcastSwapCards: %w", err)
	}
	return nil
}
