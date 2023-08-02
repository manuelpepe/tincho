package tincho

import (
	"encoding/json"
	"errors"
	"fmt"
)

type ActionType string

type Action struct {
	Type     ActionType      `json:"type"`
	Data     json.RawMessage `json:"data"`
	PlayerID string
}

const ActionStart ActionType = "start"

const ActionDraw ActionType = "draw"

type ActionDrawData struct {
	Source DrawSource `json:"source"`
}

type DrawSource string

const (
	DrawSourcePile    DrawSource = "pile"
	DrawSourceDiscard DrawSource = "discard"
)

const ActionPeekOwnCard ActionType = "effect_peek_own"

type ActionPeekOwnCardData struct {
	CardPosition int `json:"cardPosition"`
}

const ActionPeekCartaAjena ActionType = "effect_peek_carta_ajena"

type ActionPeekCartaAjenaData struct {
	CardPosition int    `json:"cardPosition"`
	Player       string `json:"player"`
}

const ActionSwapCards ActionType = "effect_swap_card"

type ActionSwapCardsData struct {
	CardPositions []int    `json:"cardPositions"`
	Players       []string `json:"players"`
}

const ActionDiscard ActionType = "discard"

type ActionDiscardData struct {
	// cardPosition = -1 means the card pending storage
	CardPosition int `json:"cardPosition"`
}

const ActionCut ActionType = "cut"

type ActionCutData struct {
	WithCount bool `json:"withCount"`
	Declared  int  `json:"declared"`
}

var ErrPendingDiscard = errors.New("someone needs to discard first")

func (r *Room) PassTurn() {
	r.CurrentTurn = (r.CurrentTurn + 1) % len(r.Players)
}

func (r *Room) broadcastPassTurn() error {
	data, err := json.Marshal(UpdateTurnData{
		Player: r.Players[r.CurrentTurn].ID,
	})
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}
	r.BroadcastUpdate(Update{
		Type: UpdateTypeTurn,
		Data: json.RawMessage(data),
	})
	return nil
}

func (r *Room) doStartGame(action Action) error {
	r.Playing = true
	if err := r.Deal(); err != nil {
		return fmt.Errorf("Deal: %w", err)
	}
	data, err := json.Marshal(UpdateStartRoundData{
		Players: r.Players,
	})
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}
	r.BroadcastUpdate(Update{
		Type: UpdateTypeStartRound,
		Data: json.RawMessage(data),
	})
	return nil
}

func (r *Room) doDraw(action Action) error {
	if r.PendingStorage != (Card{}) {
		return ErrPendingDiscard
	}
	var data ActionDrawData
	if err := json.Unmarshal(action.Data, &data); err != nil {
		return fmt.Errorf("json.Unmarshal: %w", err)
	}
	card, err := r.DrawCard(data.Source)
	if err != nil {
		return fmt.Errorf("DrawCard: %w", err)
	}
	r.PendingStorage = card
	r.PendingEffect = r.getCardEffect(card)
	if err := r.broadcastDraw(action, data); err != nil {
		return fmt.Errorf("broadcastDraw: %w", err)
	}
	return nil
}

func (r *Room) DrawCard(source DrawSource) (Card, error) {
	if len(r.DrawPile) == 0 {
		if err := r.CyclePiles(); err != nil {
			return Card{}, fmt.Errorf("ReshufflePiles: %w", err)
		}
	}
	card, err := r.drawFromSource(source)
	if err != nil {
		return Card{}, fmt.Errorf("drawFromSource: %w", err)
	}
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

func (r *Room) getCardEffect(card Card) CardEffect {
	switch card.Value {
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

func (r *Room) broadcastDraw(action Action, data ActionDrawData) error {
	mesageWithInfo, err := json.Marshal(UpdateDrawData{
		Source: data.Source,
		Card:   r.PendingStorage,
		Effect: r.PendingEffect,
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
		Effect: r.PendingEffect,
	})
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}
	r.BroadcastUpdateExcept(Update{
		Type: UpdateTypeDraw,
		Data: json.RawMessage(messageNoInfo),
	}, action.PlayerID)
	return nil
}

func (r *Room) doDiscard(action Action) error {
	var data ActionDiscardData
	if err := json.Unmarshal(action.Data, &data); err != nil {
		return fmt.Errorf("json.Unmarshal: %w", err)
	}
	if err := r.DiscardCard(action.PlayerID, data.CardPosition); err != nil {
		return fmt.Errorf("DiscardCard: %w", err)
	}
	if err := r.broadcastDiscard(action, data); err != nil {
		return fmt.Errorf("broadcastDiscard: %w", err)
	}
	r.PassTurn()
	if err := r.broadcastPassTurn(); err != nil {
		return fmt.Errorf("PassTurn: %w", err)
	}
	return nil
}

func (r *Room) DiscardCard(playerID string, card int) error {
	if card == -1 {
		r.DiscardPile = append([]Card{r.PendingStorage}, r.DiscardPile...)
		r.PendingStorage = Card{}
		r.PendingEffect = CardEffectNone
		return nil
	}
	player, exists := r.GetPlayer(playerID)
	if !exists {
		return fmt.Errorf("Unkown player: %s", playerID)
	}
	if card < -1 || card >= len(player.Hand) {
		return fmt.Errorf("invalid card position: %d", card)
	}
	r.DiscardPile = append([]Card{player.Hand[card]}, r.DiscardPile...)
	player.Hand[card] = r.PendingStorage
	r.PendingStorage = Card{}
	r.PendingEffect = CardEffectNone
	return nil
}

func (r *Room) broadcastDiscard(action Action, data ActionDiscardData) error {
	updateData, err := json.Marshal(UpdateDiscardData{
		Player:       action.PlayerID,
		CardPosition: data.CardPosition,
		Card:         r.DiscardPile[0],
	})
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}
	r.BroadcastUpdate(Update{
		Type: UpdateTypeDiscard,
		Data: json.RawMessage(updateData),
	})
	return nil
}

func (r *Room) doCut(action Action) error {
	var data ActionCutData
	if err := json.Unmarshal(action.Data, &data); err != nil {
		return fmt.Errorf("json.Unmarshal: %w", err)
	}
	player, exists := r.GetPlayer(action.PlayerID)
	if !exists {
		return fmt.Errorf("Unkown player: %s", action.PlayerID)
	}
	pointsForCutter, err := r.Cut(player, data.WithCount, data.Declared)
	if err != nil {
		return fmt.Errorf("Cut: %w", err)
	}
	r.updatePlayerPoints(player, pointsForCutter)
	if err := r.broadcastCut(player, data.WithCount, data.Declared); err != nil {
		return fmt.Errorf("broadcastCut: %w", err)
	}
	if r.IsWinConditionMet() {
		r.BroadcastUpdate(Update{Type: UpdateTypeEndGame})
	}
	return nil
}

func (r *Room) Cut(player Player, withCount bool, declared int) (int, error) {
	// check player has the lowest hand
	playerSum := player.Hand.Sum()
	for _, p := range r.Players {
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

func (r *Room) updatePlayerPoints(winner Player, pointsForWinner int) {
	for _, p := range r.Players {
		var value int
		if p.ID == winner.ID {
			value = pointsForWinner
		} else {
			value = winner.Hand.Sum()
		}
		p.Points += value
	}
}

func (r *Room) broadcastCut(player Player, withCount bool, declared int) error {
	updateData, err := json.Marshal(UpdateCutData{
		WithCount: withCount,
		Declared:  declared,
		Player:    player.ID,
		Players:   r.Players,
	})
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}
	r.BroadcastUpdate(Update{
		Type: UpdateTypeCut,
		Data: json.RawMessage(updateData),
	})
	return nil
}

func (r *Room) IsWinConditionMet() bool {
	for _, p := range r.Players {
		if p.Points > 100 {
			return true
		}
	}
	return false
}

func (r *Room) doEffectPeekOwnCard(action Action) error {
	if r.PendingEffect != CardEffectPeekOwnCard {
		return fmt.Errorf("invalid effect: %s", r.PendingEffect)
	}
	var data ActionPeekOwnCardData
	if err := json.Unmarshal(action.Data, &data); err != nil {
		return fmt.Errorf("json.Unmarshal: %w", err)
	}
	card, err := r.PeekCardAndDiscardPending(action.PlayerID, data.CardPosition)
	if err != nil {
		return fmt.Errorf("PeekCard: %w", err)
	}
	if err := r.sendPeekToPlayer(action.PlayerID, action.PlayerID, data.CardPosition, card); err != nil {
		return fmt.Errorf("broadcastDiscard: %w", err)
	}
	r.PassTurn()
	if err := r.broadcastPassTurn(); err != nil {
		return fmt.Errorf("PassTurn: %w", err)
	}
	return nil
}

func (r *Room) PeekCardAndDiscardPending(playerID string, cardIndex int) (Card, error) {
	player, exists := r.GetPlayer(playerID)
	if !exists {
		return Card{}, fmt.Errorf("Unkown player: %s", playerID)
	}
	if cardIndex < 0 || cardIndex >= len(player.Hand) {
		return Card{}, fmt.Errorf("invalid card position: %d", cardIndex)
	}
	r.DiscardPile = append([]Card{r.PendingStorage}, r.DiscardPile...)
	r.PendingStorage = Card{}
	r.PendingEffect = CardEffectNone
	return player.Hand[cardIndex], nil
}

func (r *Room) sendPeekToPlayer(targetPlayer string, peekedPlayer string, cardIndex int, card Card) error {
	updateData, err := json.Marshal(UpdatePeekCardData{
		CardPosition: cardIndex,
		Card:         card,
		Player:       peekedPlayer,
	})
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}
	r.TargetedUpdate(targetPlayer, Update{
		Type: UpdateTypePeekCard,
		Data: json.RawMessage(updateData),
	})
	return nil
}

func (r *Room) doEffectPeekCartaAjena(action Action) error {
	if r.PendingEffect != CardEffectPeekCartaAjena {
		return fmt.Errorf("invalid effect: %s", r.PendingEffect)
	}
	var data ActionPeekCartaAjenaData
	if err := json.Unmarshal(action.Data, &data); err != nil {
		return fmt.Errorf("json.Unmarshal: %w", err)
	}
	card, err := r.PeekCardAndDiscardPending(data.Player, data.CardPosition)
	if err != nil {
		return fmt.Errorf("PeekCard: %w", err)
	}
	if err := r.sendPeekToPlayer(action.PlayerID, data.Player, data.CardPosition, card); err != nil {
		return fmt.Errorf("broadcastDiscard: %w", err)
	}
	r.PassTurn()
	if err := r.broadcastPassTurn(); err != nil {
		return fmt.Errorf("PassTurn: %w", err)
	}
	return nil
}

func (r *Room) doEffectSwapCards(action Action) error {
	if r.PendingEffect != CardEffectSwapCards {
		return fmt.Errorf("invalid effect: %s", r.PendingEffect)
	}
	var data ActionSwapCardsData
	if err := json.Unmarshal(action.Data, &data); err != nil {
		return fmt.Errorf("json.Unmarshal: %w", err)
	}
	if err := r.SwapCards(data.Players, data.CardPositions); err != nil {
		return fmt.Errorf("SwapCards: %w", err)
	}
	if err := r.broadcastSwapCards(data.Players, data.CardPositions); err != nil {
		return fmt.Errorf("broadcastSwapCards: %w", err)
	}
	r.PassTurn()
	if err := r.broadcastPassTurn(); err != nil {
		return fmt.Errorf("PassTurn: %w", err)
	}
	return nil
}

func (r *Room) SwapCards(players []string, cardPositions []int) error {
	if len(players) != 2 {
		return fmt.Errorf("invalid number of players: %d", len(players))
	}
	if len(cardPositions) != 2 {
		return fmt.Errorf("invalid number of cards: %d", len(cardPositions))
	}
	player1, exists := r.GetPlayer(players[0])
	if !exists {
		return fmt.Errorf("Unkown player: %s", players[0])
	}
	player2, exists := r.GetPlayer(players[1])
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

func (r *Room) broadcastSwapCards(players []string, cardPositions []int) error {
	updateData, err := json.Marshal(UpdateSwapCardsData{
		CardPositions: cardPositions,
		Players:       players,
	})
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}
	r.BroadcastUpdate(Update{
		Type: UpdateTypeSwapCards,
		Data: json.RawMessage(updateData),
	})
	return nil
}
