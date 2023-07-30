package tincho

import "math/rand"

type Suit string

const (
	SuitSpades   Suit = "spades"
	SuitHearts   Suit = "hearts"
	SuitDiamonds Suit = "diamonds"
	SuitClubs    Suit = "clubs"
	SuitJoker    Suit = "joker"
)

type Card struct {
	Suit  Suit `json:"suit"`
	Value int  `json:"value"`
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
