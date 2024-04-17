package bots

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"slices"

	"github.com/manuelpepe/tincho/internal/game"
	"github.com/manuelpepe/tincho/internal/tincho"
)

type HardStrategy struct {
	BaseStrategy // embedded to avoid implementing all the methods

	players        []game.PlayerID
	cards          map[game.PlayerID]int
	hand           KnownHand
	firstTurn      bool
	lastDiscarded  game.Card
	lastDrawSource game.DrawSource
}

func NewHardStrategy() *HardStrategy {
	return &HardStrategy{
		players: make([]game.PlayerID, 0),
		cards:   make(map[game.PlayerID]int),
	}
}

func (s *HardStrategy) resetHand(self *tincho.Connection, players []*game.Player) {
	for _, p := range players {
		if p.ID == self.ID {
			s.hand = make(KnownHand, len(p.Hand))
			return
		}
	}
}

func (s *HardStrategy) resetPlayersHands() {
	s.cards = make(map[game.PlayerID]int)
	for _, p := range s.players {
		s.cards[p] = 4
	}
}

// TODO: Should keep state to improve performance instead of calculating every turn
func (s *HardStrategy) repeatedCards() (int, int, bool) {
	counts := make(map[int][]int)
	for ix, c := range s.hand {
		if c.IsJoker() {
			continue
		}
		if c.IsTwelveOfDiamonds() {
			continue
		}
		if _, ok := counts[c.Value]; !ok {
			counts[c.Value] = make([]int, 0)
		}
		entry := counts[c.Value]
		entry = append(entry, ix)
		counts[c.Value] = entry
	}

	for _, c := range counts {
		if len(c) > 1 {
			return c[0], c[1], true
		}
	}

	return 0, 0, false
}

func (s *HardStrategy) setPlayers(self *tincho.Connection, players []*game.Player) {
	s.players = make([]game.PlayerID, 0)
	for _, p := range players {
		if p.ID == self.ID {
			continue
		}
		s.players = append(s.players, p.ID)
	}
}

func (s *HardStrategy) getSwap() (game.PlayerID, int, game.PlayerID, int) {
	p1 := RandChoice(s.players)
	p2 := RandChoice(s.players)
	for len(s.players) > 1 && p1 == p2 {
		p2 = RandChoice(s.players)
	}
	ix1 := rand.Intn(s.cards[p1])
	ix2 := rand.Intn(s.cards[p2])
	for p1 == p2 && ix1 == ix2 && s.cards[p2] > 1 {
		ix2 = rand.Intn(s.cards[p2])
	}
	return p1, ix1, p2, ix2
}

func (s *HardStrategy) GameStart(player *tincho.Connection, data tincho.UpdateStartNextRoundData) (tincho.Action, error) {
	s.firstTurn = true
	s.lastDiscarded = data.TopDiscard
	s.resetHand(player, data.Players)
	s.setPlayers(player, data.Players)
	s.resetPlayersHands()
	return tincho.Action{Type: tincho.ActionFirstPeek}, nil
}

func (s *HardStrategy) StartNextRound(player *tincho.Connection, data tincho.UpdateStartNextRoundData) (tincho.Action, error) {
	s.firstTurn = true
	s.lastDiscarded = data.TopDiscard
	s.resetHand(player, data.Players)
	s.setPlayers(player, data.Players)
	s.resetPlayersHands()
	return tincho.Action{Type: tincho.ActionFirstPeek}, nil
}

func (s *HardStrategy) PlayerFirstPeeked(player *tincho.Connection, data tincho.UpdatePlayerFirstPeekedData) (tincho.Action, error) {
	if data.Player == player.ID {
		s.hand.Replace(0, data.Cards[0])
		s.hand.Replace(1, data.Cards[1])
	}
	return tincho.Action{}, nil
}

func (s *HardStrategy) Turn(player *tincho.Connection, data tincho.UpdateTurnData) (tincho.Action, error) {
	if data.Player != player.ID {
		return tincho.Action{}, nil
	}
	pointsInHand, knowFullHand := s.hand.KnownPoints()
	if knowFullHand && pointsInHand <= 6 {
		data, err := json.Marshal(tincho.ActionCutData{
			WithCount: true,
			Declared:  pointsInHand,
		})
		if err != nil {
			return tincho.Action{}, fmt.Errorf("json.Marshal: %w", err)
		}
		return tincho.Action{Type: tincho.ActionCut, Data: data}, nil
	} else {
		if s.lastDiscarded != (game.Card{}) {
			highestVal, found := s.hand.GetHighestValueCard()
			if found && s.hand[highestVal].Value > s.lastDiscarded.Value || s.lastDiscarded.IsJoker() || s.lastDiscarded.IsTwelveOfDiamonds() {
				s.lastDrawSource = game.DrawSourceDiscard
				data, err := json.Marshal(tincho.ActionDrawData{Source: game.DrawSourceDiscard})
				if err != nil {
					return tincho.Action{}, fmt.Errorf("json.Marshal: %w", err)
				}
				return tincho.Action{Type: tincho.ActionDraw, Data: data}, nil
			}

		}
		s.lastDrawSource = game.DrawSourcePile
		data, err := json.Marshal(tincho.ActionDrawData{Source: game.DrawSourcePile})
		if err != nil {
			return tincho.Action{}, fmt.Errorf("json.Marshal: %w", err)
		}
		s.firstTurn = false
		return tincho.Action{Type: tincho.ActionDraw, Data: data}, nil
	}
}

func (s *HardStrategy) Draw(player *tincho.Connection, data tincho.UpdateDrawData) (tincho.Action, error) {
	if data.Player != player.ID {
		return tincho.Action{}, nil
	}
	unkownCard, hasUnkownCard := s.hand.GetUnkownCard()
	if hasUnkownCard {
		if s.lastDrawSource == game.DrawSourcePile && data.Card.GetEffect() == game.CardEffectPeekOwnCard {
			res, err := json.Marshal(tincho.ActionPeekOwnCardData{CardPosition: unkownCard})
			if err != nil {
				return tincho.Action{}, fmt.Errorf("json.Marshal: %w", err)
			}
			s.hand.Replace(unkownCard, data.Card)
			return tincho.Action{Type: tincho.ActionPeekOwnCard, Data: json.RawMessage(res)}, nil
		} else if s.lastDrawSource == game.DrawSourcePile && data.Card.GetEffect() == game.CardEffectSwapCards {
			p1, c1, p2, c2 := s.getSwap()
			res, err := json.Marshal(tincho.ActionSwapCardsData{
				CardPositions: []int{c1, c2},
				Players:       []game.PlayerID{p1, p2},
			})
			if err != nil {
				return tincho.Action{}, fmt.Errorf("json.Marshal: %w", err)
			}
			return tincho.Action{Type: tincho.ActionSwapCards, Data: json.RawMessage(res)}, nil
		} else {
			res, err := json.Marshal(tincho.ActionDiscardData{CardPosition: unkownCard})
			if err != nil {
				return tincho.Action{}, fmt.Errorf("json.Marshal: %w", err)
			}
			s.hand.Replace(unkownCard, data.Card)
			return tincho.Action{Type: tincho.ActionDiscard, Data: json.RawMessage(res)}, nil
		}
	}

	// double discard if possible, doesn't have to be worth it
	if c1, c2, ok := s.repeatedCards(); ok {
		res, err := json.Marshal(tincho.ActionDiscardData{
			CardPosition:  c1,
			CardPosition2: &c2,
		})
		if err != nil {
			return tincho.Action{}, fmt.Errorf("json.Marshal: %w", err)
		}
		s.hand.Replace(c1, data.Card)
		s.hand.Remove(c2)
		return tincho.Action{Type: tincho.ActionDiscard, Data: json.RawMessage(res)}, nil
	}

	// discard highest value card
	discardIx, found := s.hand.GetHighestValueCard()
	if !found {
		// all J and 12D
		discardIx = -1
	} else if s.hand[discardIx].Value <= data.Card.Value && !data.Card.IsTwelveOfDiamonds() && !data.Card.IsJoker() {
		discardIx = -1
	}
	res, err := json.Marshal(tincho.ActionDiscardData{CardPosition: discardIx})
	if err != nil {
		return tincho.Action{}, fmt.Errorf("json.Marshal: %w", err)
	}
	if discardIx > -1 {
		s.hand.Replace(discardIx, data.Card)
	}
	return tincho.Action{Type: tincho.ActionDiscard, Data: json.RawMessage(res)}, nil
}

func (s *HardStrategy) PeekCard(player *tincho.Connection, data tincho.UpdatePeekCardData) (tincho.Action, error) {
	if data.Player != player.ID {
		return tincho.Action{}, nil
	}
	s.hand.Replace(data.CardPosition, data.Card)
	return tincho.Action{}, nil
}

func (s *HardStrategy) SwapCards(player *tincho.Connection, data tincho.UpdateSwapCardsData) (tincho.Action, error) {
	myIX := slices.Index(data.Players, player.ID)
	if myIX == -1 {
		return tincho.Action{}, nil
	}

	if myIX == 0 && data.Players[1] == player.ID {
		// swapping two from self, keep track
		c0 := s.hand[data.CardsPositions[0]]
		c1 := s.hand[data.CardsPositions[1]]
		s.hand.Replace(data.CardsPositions[0], c1)
		s.hand.Replace(data.CardsPositions[1], c0)
	} else {
		// swaping with other player, lose track
		cardPos := data.CardsPositions[myIX]
		s.hand.Forget(cardPos)
	}

	return tincho.Action{}, nil
}

func (s *HardStrategy) Discard(player *tincho.Connection, data tincho.UpdateDiscardData) (tincho.Action, error) {
	s.lastDiscarded = data.Cards[len(data.Cards)-1]
	if data.Player != player.ID {
		if len(data.CardsPositions) > 1 {
			// successful double discard
			s.cards[player.ID] -= 1
		}
	}
	return tincho.Action{}, nil
}

func (s *HardStrategy) FailedDoubleDiscard(player *tincho.Connection, data tincho.UpdateTypeFailedDoubleDiscardData) (tincho.Action, error) {
	s.lastDiscarded = data.TopOfDiscard
	if data.Player != player.ID {
		s.cards[player.ID] += 1
	}
	return tincho.Action{}, nil
}
