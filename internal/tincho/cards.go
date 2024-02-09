package tincho

import (
	"errors"
	"math/rand"
)

var ErrEmptyDeck = errors.New("empty deck")

type Suit string

const (
	SuitSpades   Suit = "spades"   // espadas
	SuitHearts   Suit = "hearts"   // copas
	SuitDiamonds Suit = "diamonds" // oro
	SuitClubs    Suit = "clubs"    // bastos
	SuitJoker    Suit = "joker"
)

type Card struct {
	Suit  Suit `json:"suit"`
	Value int  `json:"value"`
}

func (c Card) IsJoker() bool {
	return c.Suit == SuitJoker
}

func (c Card) IsTwelveOfDiamonds() bool {
	return c.Suit == SuitDiamonds && c.Value == 12
}

func (c Card) GetEffect() CardEffect {
	switch c.Value {
	case 7:
		return CardEffectPeekOwnCard
	case 8:
		return CardEffectPeekCartaAjena
	case 9:
		return CardEffectSwapCards
	default:
		return CardEffectNone
	}
}

type Deck []Card

func NewDeck() Deck {
	deck := make([]Card, 0, 50)
	for _, suit := range []Suit{SuitSpades, SuitHearts, SuitDiamonds, SuitClubs} {
		for _, value := range []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12} {
			deck = append(deck, Card{
				Suit:  suit,
				Value: value,
			})
		}
	}
	for i := 0; i < 2; i++ {
		deck = append(deck, Card{
			Suit:  SuitJoker,
			Value: 0,
		})
	}
	return deck
}

func (d *Deck) Shuffle() {
	rand.Shuffle(len(*d), func(i, j int) {
		(*d)[i], (*d)[j] = (*d)[j], (*d)[i]
	})
}

func (d *Deck) Draw() (Card, error) {
	if len(*d) == 0 {
		return Card{}, ErrEmptyDeck
	}
	card := (*d)[0]
	*d = (*d)[1:]
	return card, nil
}

type Hand []Card

func (h *Hand) Remove(pos int) {
	if h == nil {
		panic("discarding from empty hand") // TODO: maybe change to error instead of panic
	}
	*h = append((*h)[:pos], (*h)[pos+1:]...)
}

const OutOfRangeNumber = 999

// Sum returns the sum of the values of the cards in the hand.
// Rules:
//   - The 12 of diamonds is worth 0
//   - The joker value is the same as the value of the lowest card in the hand.
//   - If the hand contains only jokers, the value is 0.
//   - The rest of the cards are worth their value.
func (h Hand) Sum() int {
	min := OutOfRangeNumber
	jokers := 0
	sum := 0
	for _, card := range h {
		if card.IsTwelveOfDiamonds() {
			min = 0
			continue
		}
		if card.IsJoker() {
			jokers++
			continue
		}
		sum += card.Value
		if card.Value < min {
			min = card.Value
		}
	}
	if jokers > 0 && min < OutOfRangeNumber {
		sum += min * jokers
	}
	return sum
}

type CardEffect string

const (
	CardEffectNone           CardEffect = "none"
	CardEffectPeekOwnCard    CardEffect = "peek_own"
	CardEffectPeekCartaAjena CardEffect = "peek_carta_ajena"
	CardEffectSwapCards      CardEffect = "swap_card"
)
