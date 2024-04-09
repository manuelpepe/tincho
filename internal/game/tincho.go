package game

import (
	"errors"
	"fmt"
	"slices"
)

type DrawSource string

const (
	DrawSourcePile    DrawSource = "pile"
	DrawSourceDiscard DrawSource = "discard"
)

var ErrPendingDiscard = errors.New("someone needs to discard first")
var ErrPlayerNotPendingFirstPeek = errors.New("player not pending first peek")
var ErrPlayerAlreadyInRoom = errors.New("player already in room")
var ErrGameAlreadyStarted = errors.New("game already started")

type Round struct {
	Cutter PlayerID `json:"cutter"`

	WithCount bool `json:"withCount"`
	Declared  int  `json:"declared"`

	Scores map[PlayerID]int  `json:"scores"`
	Hands  map[PlayerID]Hand `json:"hands"`
}

type Tincho struct {
	players      []*Player
	playing      bool
	currentTurn  int
	drawPile     Deck
	discardPile  Deck
	cpyDeck      Deck
	totalRounds  int
	roundHistory []Round

	// the last card drawn that has not been stored into a player's hand
	pendingStorage Card
}

func NewTinchoWithDeck(deck Deck) *Tincho {
	return &Tincho{
		players:      make([]*Player, 0),
		playing:      false,
		drawPile:     deck,
		discardPile:  make(Deck, 0),
		cpyDeck:      slices.Clone(deck),
		totalRounds:  0,
		roundHistory: make([]Round, 0),
	}
}

func (t *Tincho) LastDiscarded() Card {
	if len(t.discardPile) == 0 {
		return Card{}
	}
	return t.discardPile[0]

}

func (t *Tincho) CountDiscardPile() int {
	return len(t.discardPile)
}

func (t *Tincho) CountDrawPile() int {
	return len(t.drawPile)
}

func (t *Tincho) GetPendingStorage() Card {
	return t.pendingStorage
}

// Playing returns whether the game has started or not. The game starts after all players complete their first peek.
func (t *Tincho) Playing() bool {
	return t.playing
}

func (t *Tincho) PlayerToPlay() *Player {
	return t.players[t.currentTurn]
}

func (t *Tincho) passTurn() {
	t.currentTurn = (t.currentTurn + 1) % len(t.players)
}

func (t *Tincho) GetPlayers() []*Player {
	return t.players
}

func (t *Tincho) GetPlayer(playerID PlayerID) (*Player, bool) {
	for _, player := range t.players {
		if player.ID == playerID {
			return player, true
		}
	}
	return nil, false
}

func (t *Tincho) AddPlayer(p *Player) error {
	if t.playing {
		return ErrGameAlreadyStarted
	}
	if _, exists := t.GetPlayer(p.ID); exists {
		return ErrPlayerAlreadyInRoom
	}
	t.players = append(t.players, p)
	return nil
}

// StartGame starts the game by setting all players to pending first peek and dealing 4 cards to each player.
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
		for i := 0; i < 4; i++ {
			card, err := t.drawPile.Draw()
			if err != nil {
				return err
			}
			t.players[pid].Hand = append(t.players[pid].Hand, card)
		}
	}
	return nil
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
	if len(t.drawPile) == 0 {
		if err := t.cyclePiles(); err != nil {
			return Card{}, fmt.Errorf("CyclePiles: %w", err)
		}
	}
	card, err := t.drawFromSource(source)
	if err != nil {
		return Card{}, fmt.Errorf("drawFromSource: %w", err)
	}
	t.pendingStorage = card
	return card, nil
}

func (r *Tincho) cyclePiles() error {
	r.drawPile = r.discardPile
	r.drawPile.Shuffle()
	r.discardPile = make(Deck, 0)
	return r.discardTopCard()
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

// Discard a card after drawing.
// If position is -1, the drawn card is discarded without storing it in the player's hand.
// If position is a valid hand index, the drawn card is stored in the player's hand at the given position
// and the card at that position discarded.
// After discarding the turn passes to the next player.
func (t *Tincho) Discard(position int) (DiscardedCard, error) {
	if t.pendingStorage == (Card{}) {
		return Card{}, errors.New("can't discard without drawing")
	}

	player := t.players[t.currentTurn]
	if position < -1 || position >= len(player.Hand) {
		return Card{}, fmt.Errorf("invalid card position: %d", position)
	}

	if position == -1 {
		t.discardPile = append([]Card{t.pendingStorage}, t.discardPile...)
	} else {
		t.discardPile = append([]Card{player.Hand[position]}, t.discardPile...)
		player.Hand[position] = t.pendingStorage
	}

	t.pendingStorage = Card{}
	t.passTurn()

	return t.discardPile[0], nil
}

// Discard two cards after drawing.
// The cards must be equals, otherwise the double discard fails with an ErrDiscardingNonEqualCards value.
// If the discard fails, the top card of the discard pile is returned in the second return value as it must be
// drawn, sometimes from a freshly shuffled draw pile.
func (t *Tincho) DiscardTwo(position int, position2 int) ([]DiscardedCard, DiscardedCard, error) {
	if t.pendingStorage == (Card{}) {
		return nil, Card{}, errors.New("can't discard without drawing")
	}
	cards, topCardOnFail, err := t.discardTwoCards(position, position2)
	if err != nil {
		return cards, topCardOnFail, fmt.Errorf("error discarding: %w", err)
	}
	t.passTurn()
	return cards, Card{}, nil
}

var ErrDiscardingNonEqualCards = errors.New("tried to double discard cards of different values")

// Try to discard two cards from the player's hand.
// Both positions must be different and from the player's hand (drawn card can't be doble discarded).
// Both cards must be of the same value, jokers can't be paired with non joker cards.
func (t *Tincho) discardTwoCards(position1 int, position2 int) ([]DiscardedCard, DiscardedCard, error) {
	player := t.players[t.currentTurn]
	if position1 == position2 {
		return nil, Card{}, fmt.Errorf("invalid card positions: %d, %d", position1, position2)
	}
	if position1 < 0 || position1 >= len(player.Hand) {
		return nil, Card{}, fmt.Errorf("invalid card position: %d", position1)
	}
	if position2 < 0 || position2 >= len(player.Hand) {
		return nil, Card{}, fmt.Errorf("invalid card position: %d", position2)
	}

	card1 := player.Hand[position1]
	card2 := player.Hand[position2]

	if card1.Value != card2.Value {
		// draw a new card if discard pile is empty
		if len(t.discardPile) == 0 {
			if err := t.discardTopCard(); err != nil {
				return nil, Card{}, fmt.Errorf("discardTopCard: %w", err)
			}
		}

		// player keeps all 3 cards in hand
		player.Hand = append(player.Hand, t.pendingStorage)
		t.pendingStorage = Card{}
		return []Card{card1, card2}, t.discardPile[0], ErrDiscardingNonEqualCards
	}

	// player succesfully discards both cards
	t.discardPile = append([]Card{card1, card2}, t.discardPile...)
	player.Hand[position1] = t.pendingStorage
	player.Hand.Remove(position2)
	t.pendingStorage = Card{}
	return []Card{card1, card2}, Card{}, nil
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

func (t *Tincho) IsWinConditionMet() bool {
	for _, p := range t.players {
		if p.Points > 100 {
			return true
		}
	}
	return false
}

type PeekedCard = Card

// A card to be in the discard pile
type DiscardedCard = Card

func (t *Tincho) UseEffectPeekOwnCard(position int) (PeekedCard, DiscardedCard, error) {
	if t.pendingStorage.GetEffect() != CardEffectPeekOwnCard {
		return Card{}, Card{}, fmt.Errorf("invalid effect: %s", t.pendingStorage.GetEffect())
	}
	player := t.players[t.currentTurn]
	card, err := t.peekCard(player, position)
	if err != nil {
		return Card{}, Card{}, fmt.Errorf("PeekCard: %w", err)
	}
	discarded := t.discardPending()
	t.passTurn()
	return card, discarded, nil
}

func (t *Tincho) UseEffectPeekCartaAjena(playerID PlayerID, position int) (PeekedCard, DiscardedCard, error) {
	if t.pendingStorage.GetEffect() != CardEffectPeekCartaAjena {
		return Card{}, Card{}, fmt.Errorf("invalid effect: %s", t.pendingStorage.GetEffect())
	}
	player, ok := t.GetPlayer(playerID)
	if !ok {
		return Card{}, Card{}, fmt.Errorf("player not found: %s", playerID)
	}
	card, err := t.peekCard(player, position)
	if err != nil {
		return Card{}, Card{}, fmt.Errorf("PeekCard: %w", err)
	}
	discarded := t.discardPending()
	t.passTurn()
	return card, discarded, nil
}

func (t *Tincho) peekCard(player *Player, cardIndex int) (PeekedCard, error) {
	if cardIndex < 0 || cardIndex >= len(player.Hand) {
		return Card{}, fmt.Errorf("invalid card position: %d", cardIndex)
	}
	return player.Hand[cardIndex], nil
}

func (t *Tincho) UseEffectSwapCards(players []PlayerID, positions []int) (DiscardedCard, error) {
	if t.pendingStorage.GetEffect() != CardEffectSwapCards {
		return Card{}, fmt.Errorf("invalid effect: %s", t.pendingStorage.GetEffect())
	}
	if err := t.swapCards(players, positions); err != nil {
		return Card{}, fmt.Errorf("SwapCards: %w", err)
	}
	discarded := t.discardPending()
	t.passTurn()
	return discarded, nil
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
