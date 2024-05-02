package tincho

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/manuelpepe/tincho/pkg/game"
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

type ActionData interface {
	ActionDrawData |
		ActionPeekOwnCardData |
		ActionPeekCartaAjenaData |
		ActionSwapCardsData |
		ActionDiscardData |
		ActionCutData |
		ActionWithoutData
}

// TypedAction is an interface used to pass around Action[T] types without needing to
// know the exact type of T. Do not implement this interface, use Action[T] instead.
type TypedAction interface {
	GetType() ActionType
	SetPlayerID(game.PlayerID)
	GetPlayerID() game.PlayerID
}

type Action[T ActionData] struct {
	Type     ActionType `json:"type"`
	Data     T          `json:"data"`
	PlayerID game.PlayerID
}

func (a *Action[T]) GetType() ActionType {
	if a == nil {
		return ""
	}
	return a.Type
}

func (a *Action[T]) SetPlayerID(playerID game.PlayerID) {
	a.PlayerID = playerID
}

func (a *Action[T]) GetPlayerID() game.PlayerID {
	if a == nil {
		return "<UNSET>"
	}
	return a.PlayerID
}

func NewActionFromRawMessage(message []byte) (TypedAction, error) {
	var actionType struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(message, &actionType); err != nil {
		return nil, err
	}
	var action TypedAction
	switch actionType.Type {
	case string(ActionStart):
		action = &Action[ActionWithoutData]{Type: ActionStart}
	case string(ActionFirstPeek):
		action = &Action[ActionWithoutData]{Type: ActionFirstPeek}
	case string(ActionDraw):
		var act Action[ActionDrawData]
		if err := json.Unmarshal(message, &act); err != nil {
			return nil, err
		}
		action = &act
	case string(ActionPeekOwnCard):
		var act Action[ActionPeekOwnCardData]
		if err := json.Unmarshal(message, &act); err != nil {
			return nil, err
		}
		action = &act
	case string(ActionPeekCartaAjena):
		var act Action[ActionPeekCartaAjenaData]
		if err := json.Unmarshal(message, &act); err != nil {
			return nil, err
		}
		action = &act
	case string(ActionSwapCards):
		var act Action[ActionSwapCardsData]
		if err := json.Unmarshal(message, &act); err != nil {
			return nil, err
		}
		action = &act
	case string(ActionDiscard):
		var act Action[ActionDiscardData]
		if err := json.Unmarshal(message, &act); err != nil {
			return nil, err
		}
		action = &act
	case string(ActionCut):
		var act Action[ActionCutData]
		if err := json.Unmarshal(message, &act); err != nil {
			return nil, err
		}
		action = &act
	default:
		return nil, fmt.Errorf("unknown action type: %s", actionType.Type)
	}

	return action, nil
}

type Game_DrawSource string

type ActionWithoutData struct{}

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

func (r *Room) doStartGame(action Action[ActionWithoutData]) error {
	if r.state.GetPlayers()[0].ID != action.PlayerID {
		return ErrNotRoomLeader
	}
	if err := r.broadcastGameConfig(r.state.CountBaseDeck()); err != nil {
		return fmt.Errorf("broadcastGameConfig: %w", err)
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

func (r *Room) doPeekTwo(action Action[ActionWithoutData]) error {
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

func (r *Room) doDraw(action Action[ActionDrawData]) error {
	card, err := r.state.Draw(action.Data.Source)
	if err != nil {
		return err
	}
	if err := r.broadcastDraw(action.PlayerID, action.Data.Source, card); err != nil {
		return fmt.Errorf("broadcastDraw: %w", err)
	}
	return nil
}

func (r *Room) doDiscard(action Action[ActionDiscardData]) error {
	var positions []int
	var cycledPiles game.CycledPiles
	var values []game.Card
	var err error

	data := action.Data

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

func (r *Room) doCut(action Action[ActionCutData]) error {
	data := action.Data
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
		r.close()
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

func (r *Room) doEffectPeekOwnCard(action Action[ActionPeekOwnCardData]) error {
	card, discarded, cycledPiles, err := r.state.UseEffectPeekOwnCard(action.Data.CardPosition)
	if err != nil {
		return err
	}
	err = r.broadcastPeek(action.PlayerID, action.PlayerID, action.Data.CardPosition, card, discarded, cycledPiles)
	if err != nil {
		return fmt.Errorf("broadcastDiscard: %w", err)
	}
	return nil
}

func (r *Room) doEffectPeekCartaAjena(action Action[ActionPeekCartaAjenaData]) error {
	card, discarded, cycledPiles, err := r.state.UseEffectPeekCartaAjena(action.Data.Player, action.Data.CardPosition)
	if err != nil {
		return err
	}
	err = r.broadcastPeek(action.PlayerID, action.Data.Player, action.Data.CardPosition, card, discarded, cycledPiles)
	if err != nil {
		return fmt.Errorf("broadcastDiscard: %w", err)
	}
	return nil
}

func (r *Room) doEffectSwapCards(action Action[ActionSwapCardsData]) error {
	discarded, cycledPiles, err := r.state.UseEffectSwapCards(action.Data.Players, action.Data.CardPositions)
	if err != nil {
		return err
	}
	err = r.broadcastSwapCards(action.PlayerID, action.Data.CardPositions, action.Data.Players, discarded, cycledPiles)
	if err != nil {
		return fmt.Errorf("broadcastSwapCards: %w", err)
	}
	return nil
}
