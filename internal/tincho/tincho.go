package tincho

import (
	"errors"
	"fmt"
)

var ErrPendingDiscard = errors.New("someone needs to discard first")
var ErrPlayerNotPendingFirstPeek = errors.New("player not pending first peek")
var ErrPlayerAlreadyInRoom = errors.New("player already in room")
var ErrGameAlreadyStarted = errors.New("game already started")

type Tincho struct {
	players     []Player
	playing     bool
	currentTurn int
	drawPile    Deck
	discardPile Deck

	// the last card drawn that has not been stored into a player's hand
	pendingStorage Card
}

func NewTinchoWithDeck(deck Deck) *Tincho {
	return &Tincho{
		players:     make([]Player, 0),
		playing:     false,
		drawPile:    deck,
		discardPile: make(Deck, 0),
	}
}

// Playing returns whether the game has started or not. The game starts after all players complete their first peek.
func (t *Tincho) Playing() bool {
	return t.playing
}

func (t *Tincho) PlayerToPlay() Player {
	return t.players[t.currentTurn]
}

func (t *Tincho) passTurn() {
	t.currentTurn = (t.currentTurn + 1) % len(t.players)
}

func (t *Tincho) GetPlayers() []Player {
	return t.players
}

func (t *Tincho) getPlayer(playerID string) (Player, bool) {
	for _, room := range t.players {
		if room.ID == playerID {
			return room, true
		}
	}
	return Player{}, false
}

func (t *Tincho) AddPlayer(p Player) error {
	if t.playing {
		return ErrGameAlreadyStarted
	}
	if _, exists := t.getPlayer(p.ID); exists {
		return ErrPlayerAlreadyInRoom
	}
	t.players = append(t.players, p)
	return nil
}

// StartGame starts the game by setting all players to pending first peek and dealing 4 cards to each player.
func (t *Tincho) StartGame() error {
	t.setAllPlayersPendingFirstPeek()
	if err := t.deal(); err != nil {
		return fmt.Errorf("Deal: %w", err)
	}
	return nil
}

func (r *Tincho) setAllPlayersPendingFirstPeek() {
	for i := range r.players {
		r.players[i].PendingFirstPeek = true
	}
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

// GetFirstPeek allows to peek two cards from a players hand if it hasn't peeked yet.
func (t *Tincho) GetFirstPeek(playerID string) ([]Card, error) {
	player, exists := t.getPlayer(playerID)
	if !exists {
		return nil, fmt.Errorf("Unkown player: %s", playerID)
	}
	if !player.PendingFirstPeek {
		return nil, fmt.Errorf("%w: %s", ErrPlayerNotPendingFirstPeek, playerID)
	}
	var peekedCards []Card
	for _, position := range []int{0, 1} {
		peekedCards = append(peekedCards, player.Hand[position])
	}
	t.setPlayerFirstPeekDone(playerID)
	if t.AllPlayersFirstPeeked() {
		t.playing = true
	}
	return peekedCards, nil
}

func (r *Tincho) setPlayerFirstPeekDone(player string) {
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

func (r *Tincho) discardTopCard() error {
	card, err := r.drawPile.Draw()
	if err != nil {
		return err
	}
	r.discardPile = append(r.discardPile, card)
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

func (t *Tincho) Discard(position int, position2 *int) ([]Card, error) {
	if position2 == nil {
		return t.discardOneCard(position)
	} else {
		return t.discardTwoCards(position, *position2)
	}
}

var ErrDiscardingNonEqualCards = errors.New("tried to double discard cards of different values")

// Try to discard two cards from the player's hand. Both positions must be different and from the player's hand (drawn card can't be doble discarded).
// Both cards must be of the same value, jokers can't be paired with non joker cards.
func (t *Tincho) discardTwoCards(position1 int, position2 int) ([]Card, error) {
	player := t.players[t.currentTurn]
	if position1 == position2 {
		return nil, fmt.Errorf("invalid card positions: %d, %d", position1, position2)
	}
	if position1 < 0 || position1 >= len(player.Hand) {
		return nil, fmt.Errorf("invalid card position: %d", position1)
	}
	if position2 < 0 || position2 >= len(player.Hand) {
		return nil, fmt.Errorf("invalid card position: %d", position2)
	}
	card1 := player.Hand[position1]
	card2 := player.Hand[position2]

	if card1.Value != card2.Value {
		// Player keeps all 3 cards in hand
		player.Hand = append(player.Hand, t.pendingStorage)
		t.pendingStorage = Card{}
		return []Card{card1, card2}, ErrDiscardingNonEqualCards
	}

	t.discardPile = append([]Card{card1, card2}, t.discardPile...) // discard both cards
	player.Hand[position1] = t.pendingStorage
	player.Hand.Remove(position2)
	t.pendingStorage = Card{}

	t.passTurn()
	return []Card{card1, card2}, nil
}

// Discard a card after drawing. If position is -1, the card is discarded without storing it in the player's hand.
// Otherwise, the card is stored in the player's hand at the given position.
// After discarding the turn passes to the next player.
func (t *Tincho) discardOneCard(position int) ([]Card, error) {
	player := t.players[t.currentTurn]
	card, err := t.discardCard(player, position)
	if err != nil {
		return nil, fmt.Errorf("discardCard: %w", err)
	}
	t.passTurn()
	return []Card{card}, nil
}

func (t *Tincho) discardCard(player Player, card int) (Card, error) {
	if card < -1 || card >= len(player.Hand) {
		return Card{}, fmt.Errorf("invalid card position: %d", card)
	}
	if card == -1 {
		t.discardPile = append([]Card{t.pendingStorage}, t.discardPile...)
	} else {
		t.discardPile = append([]Card{player.Hand[card]}, t.discardPile...)
		player.Hand[card] = t.pendingStorage
	}
	t.pendingStorage = Card{}
	return t.discardPile[0], nil
}

// Cut finishes the current round and updates the points for all players.
func (t *Tincho) Cut(withCount bool, declared int) error {
	player := t.players[t.currentTurn]
	pointsForCutter, err := t.cut(player, withCount, declared)
	if err != nil {
		return fmt.Errorf("Cut: %w", err)
	}
	t.updatePlayerPoints(player, pointsForCutter)
	if t.IsWinConditionMet() {
		t.playing = false
	}
	return nil
}

func (t *Tincho) cut(player Player, withCount bool, declared int) (int, error) {
	// check player has the lowest hand
	playerSum := player.Hand.Sum()
	for _, p := range t.players {
		if p.ID != player.ID && p.Hand.Sum() <= playerSum {
			return playerSum + 20, nil // absolute fail
		}
	}
	if !withCount {
		return 0, nil // wins
	}
	if declared == playerSum {
		return -10, nil // wins + bonus
	}
	return playerSum + 10, nil // loss + bonus
}

func (t *Tincho) updatePlayerPoints(winner Player, pointsForWinner int) {
	for ix := range t.players {
		var value int
		if t.players[ix].ID == winner.ID {
			value = pointsForWinner
		} else {
			value = winner.Hand.Sum()
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

func (t *Tincho) UseEffectPeekOwnCard(position int) (Card, error) {
	if t.pendingStorage.GetEffect() != CardEffectPeekOwnCard {
		return Card{}, fmt.Errorf("invalid effect: %s", t.pendingStorage.GetEffect())
	}
	player := &t.players[t.currentTurn]
	card, err := t.peekCardAndDiscardPending(player, position)
	if err != nil {
		return Card{}, fmt.Errorf("PeekCard: %w", err)
	}
	t.passTurn()
	return card, nil
}

func (t *Tincho) UseEffectPeekCartaAjena(position int) (Card, error) {
	if t.pendingStorage.GetEffect() != CardEffectPeekCartaAjena {
		return Card{}, fmt.Errorf("invalid effect: %s", t.pendingStorage.GetEffect())
	}
	player := &t.players[t.currentTurn]
	card, err := t.peekCardAndDiscardPending(player, position)
	if err != nil {
		return Card{}, fmt.Errorf("PeekCard: %w", err)
	}
	t.passTurn()
	return card, nil
}

func (t *Tincho) peekCardAndDiscardPending(player *Player, cardIndex int) (Card, error) {
	if cardIndex < 0 || cardIndex >= len(player.Hand) {
		return Card{}, fmt.Errorf("invalid card position: %d", cardIndex)
	}
	t.discardPile = append([]Card{t.pendingStorage}, t.discardPile...)
	t.pendingStorage = Card{}
	return player.Hand[cardIndex], nil
}

func (t *Tincho) UseEffectSwapCards(players []string, positions []int) error {
	if t.pendingStorage.GetEffect() != CardEffectSwapCards {
		return fmt.Errorf("invalid effect: %s", t.pendingStorage.GetEffect())
	}
	if err := t.swapCards(players, positions); err != nil {
		return fmt.Errorf("SwapCards: %w", err)
	}
	t.passTurn()
	return nil
}

func (t *Tincho) swapCards(players []string, cardPositions []int) error {
	if len(players) != 2 {
		return fmt.Errorf("invalid number of players: %d", len(players))
	}
	if len(cardPositions) != 2 {
		return fmt.Errorf("invalid number of cards: %d", len(cardPositions))
	}
	player1, exists := t.getPlayer(players[0])
	if !exists {
		return fmt.Errorf("Unkown player: %s", players[0])
	}
	player2, exists := t.getPlayer(players[1])
	if !exists {
		return fmt.Errorf("Unkown player: %s", players[1])
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
