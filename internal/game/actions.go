package game

import (
	"errors"
	"fmt"
	"slices"
)

// StartGame starts the game by setting all players to pending first peek and dealing STARTING_HAND_SIZE (4) cards to each player.
func (t *Tincho) StartGame() (Card, error) {
	if t.playing {
		return Card{}, ErrGameAlreadyStarted
	}
	t.playing = true
	return t.prepareForNextRound(false)
}

func (t *Tincho) StartNextRound() (Card, error) {
	if !t.playing {
		return Card{}, fmt.Errorf("game not started")
	}
	topDiscard, err := t.prepareForNextRound(true)
	if err != nil {
		return Card{}, fmt.Errorf("prepareForNextRound: %w", err)
	}
	return topDiscard, nil
}

func (t *Tincho) prepareForNextRound(shuffleDeck bool) (Card, error) {
	t.totalRounds += 1
	t.currentTurn = (t.totalRounds - 1) % len(t.players)
	for i := range t.players {
		t.players[i].PendingFirstPeek = true
		t.players[i].Hand = make(Hand, 0)
	}
	t.pendingStorage = Card{}
	t.discardPile = make(Deck, 0)
	t.drawPile = slices.Clone(t.cpyDeck)
	if shuffleDeck {
		t.drawPile.Shuffle()
	}
	if err := t.deal(); err != nil {
		return Card{}, fmt.Errorf("deal: %w", err)
	}
	if err := t.discardTopCard(); err != nil {
		return Card{}, fmt.Errorf("discardTopCard: %w", err)
	}
	return t.discardPile[0], nil
}

func (t *Tincho) deal() error {
	for pid := range t.players {
		for i := 0; i < STARTING_HAND_SIZE; i++ {
			card, err := t.drawPile.Draw()
			if err != nil {
				return err
			}
			t.players[pid].Hand = append(t.players[pid].Hand, card)
		}
	}
	return nil
}

// Sends the top card in the draw pile to the discard pile.
func (r *Tincho) discardTopCard() error {
	card, err := r.drawPile.Draw()
	if err != nil {
		return err
	}
	r.discardPile = append([]Card{card}, r.discardPile...)
	return nil
}

// GetFirstPeek allows to peek two cards from a players hand if it hasn't peeked yet.
func (t *Tincho) GetFirstPeek(playerID PlayerID) ([]Card, error) {
	player, exists := t.GetPlayer(playerID)
	if !exists {
		return nil, fmt.Errorf("unkown player: %s", playerID)
	}
	if !player.PendingFirstPeek {
		return nil, fmt.Errorf("%w: %s", ErrPlayerNotPendingFirstPeek, playerID)
	}
	var peekedCards []Card
	for _, position := range []int{0, 1} {
		peekedCards = append(peekedCards, player.Hand[position])
	}
	t.setPlayerFirstPeekDone(playerID)
	return peekedCards, nil
}

func (r *Tincho) setPlayerFirstPeekDone(player PlayerID) {
	for i := range r.players {
		if r.players[i].ID == player {
			r.players[i].PendingFirstPeek = false
			return
		}
	}
}

func (r *Tincho) AllPlayersFirstPeeked() bool {
	for _, p := range r.players {
		if p.PendingFirstPeek {
			return false
		}
	}
	return true
}

// Draw grabs a card from the given source. Grabbing a card means that it is not yet in the player's
// hand, but it is stored in the PendingStorage field. A following call to Discard will store the card
// in the player's hand or discard it.
func (t *Tincho) Draw(source DrawSource) (Card, error) {
	if t.pendingStorage != (Card{}) {
		return Card{}, ErrPendingDiscard
	}
	card, err := t.drawFromSource(source)
	if err != nil {
		return Card{}, fmt.Errorf("drawFromSource: %w", err)
	}
	t.pendingStorage = card
	return card, nil
}

func (t *Tincho) drawFromSource(source DrawSource) (Card, error) {
	switch source {
	case DrawSourcePile:
		return t.drawPile.Draw()
	case DrawSourceDiscard:
		return t.discardPile.Draw()
	default:
		return Card{}, fmt.Errorf("invalid source: %s", source)
	}
}

func (t *Tincho) cyclePilesIfEmptyDraw() CycledPiles {
	cycledPiles := len(t.drawPile) == 0
	if cycledPiles {
		last_discarded, err := t.discardPile.Draw()
		t.drawPile = t.discardPile
		t.drawPile.Shuffle()
		t.discardPile = make(Deck, 0)
		if err == nil {
			t.discardPile = append([]Card{last_discarded}, t.discardPile...)
		}
	}
	return CycledPiles(cycledPiles)
}

// Wether the discard pile has been shuffled into the draw pile
type CycledPiles bool

// Discard a card after drawing.
// If position is -1, the drawn card is discarded without storing it in the player's hand.
// If position is a valid hand index, the drawn card is stored in the player's hand at the given position
// and the card at that position discarded.
// After discarding the turn passes to the next player.
func (t *Tincho) Discard(position int) (DiscardedCard, CycledPiles, error) {
	if t.pendingStorage == (Card{}) {
		return Card{}, false, errors.New("can't discard without drawing")
	}

	player := t.players[t.currentTurn]
	if position < -1 || position >= len(player.Hand) {
		return Card{}, false, fmt.Errorf("invalid card position: %d", position)
	}

	if position == -1 {
		t.discardPile = append([]Card{t.pendingStorage}, t.discardPile...)
	} else {
		t.discardPile = append([]Card{player.Hand[position]}, t.discardPile...)
		player.Hand[position] = t.pendingStorage
	}

	t.pendingStorage = Card{}
	cycledPiles := t.cyclePilesIfEmptyDraw()
	t.passTurn()

	return t.discardPile[0], cycledPiles, nil
}

// Discard two cards after drawing.
// The cards must be equals, otherwise the double discard fails with an ErrDiscardingNonEqualCards value.
// If the discard fails, the top card of the discard pile is returned in the second return value as it must be
// drawn, sometimes from a freshly shuffled draw pile.
func (t *Tincho) DiscardTwo(position int, position2 int) ([]DiscardedCard, DiscardedCard, CycledPiles, error) {
	if t.pendingStorage == (Card{}) {
		return nil, Card{}, false, errors.New("can't discard without drawing")
	}

	cards, topCardOnFail, cycledPiles, err := t.discardTwoCards(position, position2)
	if err != nil {
		if errors.Is(err, ErrDiscardingNonEqualCards) {
			t.passTurn()
		}
		return cards, topCardOnFail, cycledPiles, fmt.Errorf("error discarding: %w", err)
	}

	t.passTurn()
	return cards, Card{}, cycledPiles, nil
}

var ErrDiscardingNonEqualCards = errors.New("tried to double discard cards of different values")

// Try to discard two cards from the player's hand.
// Both positions must be different and from the player's hand (drawn card can't be doble discarded).
// Both cards must be of the same value, jokers can't be paired with non joker cards.
func (t *Tincho) discardTwoCards(position1 int, position2 int) ([]DiscardedCard, DiscardedCard, CycledPiles, error) {
	player := t.players[t.currentTurn]
	if position1 == position2 {
		return nil, Card{}, false, fmt.Errorf("invalid card positions: %d, %d", position1, position2)
	}
	if position1 < 0 || position1 >= len(player.Hand) {
		return nil, Card{}, false, fmt.Errorf("invalid card position: %d", position1)
	}
	if position2 < 0 || position2 >= len(player.Hand) {
		return nil, Card{}, false, fmt.Errorf("invalid card position: %d", position2)
	}

	cycledPiles := t.cyclePilesIfEmptyDraw()

	card1 := player.Hand[position1]
	card2 := player.Hand[position2]

	if card1.Value != card2.Value {
		// player keeps all 3 cards in hand
		player.Hand = append(player.Hand, t.pendingStorage)
		t.pendingStorage = Card{}
		if len(t.discardPile) == 0 {
			t.discardTopCard()
		}
		return []Card{card1, card2}, t.discardPile[0], cycledPiles, ErrDiscardingNonEqualCards
	}

	// player succesfully discards both cards
	t.discardPile = append([]Card{card1, card2}, t.discardPile...)
	player.Hand[position1] = t.pendingStorage
	player.Hand.Remove(position2)
	t.pendingStorage = Card{}
	return []Card{card1, card2}, Card{}, cycledPiles, nil
}

type GameFinished bool

// Cut finishes the current round and updates the points for all players.
func (t *Tincho) Cut(withCount bool, declared int) ([]Round, GameFinished, error) {
	player := t.players[t.currentTurn]
	t.updatePlayerPoints(player, withCount, declared)
	t.recordScores(player.ID, withCount, declared)
	if t.IsWinConditionMet() {
		t.playing = false
	}
	return t.roundHistory, GameFinished(!t.playing), nil
}

func (t *Tincho) recordScores(cutter PlayerID, withCount bool, declared int) {
	round := Round{
		Cutter:    cutter,
		WithCount: withCount,
		Declared:  declared,
		Scores:    make(map[PlayerID]int),
		Hands:     make(map[PlayerID]Hand),
	}
	for _, p := range t.players {
		round.Scores[p.ID] = p.Points
		round.Hands[p.ID] = p.Hand
	}
	t.roundHistory = append(t.roundHistory, round)
}

func (t *Tincho) calculatePointsForCutter(cutter *Player, withCount bool, declared int) int {
	// check player has the lowest hand
	playerSum := cutter.Hand.Sum()
	for _, p := range t.players {
		if p.ID != cutter.ID && p.Hand.Sum() <= playerSum {
			return playerSum + 20 // absolute fail
		}
	}
	if !withCount {
		return 0 // wins
	}
	if declared == playerSum {
		return -10 // wins + bonus
	}
	return playerSum + 10 // loss + bonus
}

func (t *Tincho) updatePlayerPoints(cutter *Player, withCount bool, declared int) {
	for ix := range t.players {
		var value int
		if t.players[ix].ID == cutter.ID {
			value = t.calculatePointsForCutter(cutter, withCount, declared)
		} else {
			value = t.players[ix].Hand.Sum()
		}
		t.players[ix].Points += value
	}
}

// A card that has been peeked
type PeekedCard = Card

// A card to be in the discard pile
type DiscardedCard = Card

func (t *Tincho) UseEffectPeekOwnCard(position int) (PeekedCard, DiscardedCard, CycledPiles, error) {
	if t.pendingStorage.GetEffect() != CardEffectPeekOwnCard {
		return Card{}, Card{}, false, fmt.Errorf("invalid effect: %s", t.pendingStorage.GetEffect())
	}

	player := t.players[t.currentTurn]
	card, err := t.peekCard(player, position)
	if err != nil {
		return Card{}, Card{}, false, fmt.Errorf("PeekCard: %w", err)
	}

	discarded := t.discardPending()
	cycledPiles := t.cyclePilesIfEmptyDraw()
	t.passTurn()
	return card, discarded, cycledPiles, nil
}

func (t *Tincho) UseEffectPeekCartaAjena(playerID PlayerID, position int) (PeekedCard, DiscardedCard, CycledPiles, error) {
	if t.pendingStorage.GetEffect() != CardEffectPeekCartaAjena {
		return Card{}, Card{}, false, fmt.Errorf("invalid effect: %s", t.pendingStorage.GetEffect())
	}

	player, ok := t.GetPlayer(playerID)
	if !ok {
		return Card{}, Card{}, false, fmt.Errorf("player not found: %s", playerID)
	}

	card, err := t.peekCard(player, position)
	if err != nil {
		return Card{}, Card{}, false, fmt.Errorf("PeekCard: %w", err)
	}

	discarded := t.discardPending()
	cycledPiles := t.cyclePilesIfEmptyDraw()
	t.passTurn()
	return card, discarded, cycledPiles, nil
}

func (t *Tincho) peekCard(player *Player, cardIndex int) (PeekedCard, error) {
	if cardIndex < 0 || cardIndex >= len(player.Hand) {
		return Card{}, fmt.Errorf("invalid card position: %d", cardIndex)
	}
	return player.Hand[cardIndex], nil
}

func (t *Tincho) UseEffectSwapCards(players []PlayerID, positions []int) (DiscardedCard, CycledPiles, error) {
	if t.pendingStorage.GetEffect() != CardEffectSwapCards {
		return Card{}, false, fmt.Errorf("invalid effect: %s", t.pendingStorage.GetEffect())
	}

	if err := t.swapCards(players, positions); err != nil {
		return Card{}, false, fmt.Errorf("SwapCards: %w", err)
	}

	discarded := t.discardPending()
	cycledPiles := t.cyclePilesIfEmptyDraw()
	t.passTurn()
	return discarded, cycledPiles, nil
}

func (t *Tincho) swapCards(players []PlayerID, cardPositions []int) error {
	if len(players) != 2 {
		return fmt.Errorf("invalid number of players: %d", len(players))
	}
	if len(cardPositions) != 2 {
		return fmt.Errorf("invalid number of cards: %d", len(cardPositions))
	}
	player1, exists := t.GetPlayer(players[0])
	if !exists {
		return fmt.Errorf("unkown player: %s", players[0])
	}
	player2, exists := t.GetPlayer(players[1])
	if !exists {
		return fmt.Errorf("unkown player: %s", players[1])
	}
	if cardPositions[0] < 0 || cardPositions[0] >= len(player1.Hand) {
		return fmt.Errorf("invalid card position: %d", cardPositions[0])
	}
	if cardPositions[1] < 0 || cardPositions[1] >= len(player2.Hand) {
		return fmt.Errorf("invalid card position: %d", cardPositions[1])
	}
	player1.Hand[cardPositions[0]], player2.Hand[cardPositions[1]] = player2.Hand[cardPositions[1]], player1.Hand[cardPositions[0]]
	return nil
}

func (t *Tincho) discardPending() DiscardedCard {
	cpy := t.pendingStorage
	t.discardPile = append([]Card{t.pendingStorage}, t.discardPile...)
	t.pendingStorage = Card{}
	return cpy
}
