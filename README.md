# Tincholi

A multiplayer card game for the whole family.

## Description

Each player is dealt 4 cards face down. In turns, each player draws a card from the deck and decides between replacing a card from their hand with it or discarding the drawn card. Some cards have special effects (peek your own card, an oponents card or swap any two cards). In their turn, a player may decide to cut instead of drawing a card. Additionally, the player can choose to *blindly* call out their hand total. At this point, all cards are turned face up and each player adds the sum of their cards to their score. The cutting player is behold to some extra rules that decide if they deduct points from their score, or get more points of top of their hand (check cutting rules carefully). The first player to get a score over 100 loses, ending the game.

### Cutting rules

The cutting player must have the lowest hand in the table, otherwise they instanly lose and must add their hand total + 20 points to their score (add 20 + hand to score this round).

If the cutting player has the lowest hand and decided to not call their total, their score this round is 0.

If the cutting player has the lowest hand and called their total right, their score is -10.

If the cutting player has the lowest hand and called their total wrong, their score is hand total + 10.

All other players only add their hand totals to their score.

## Running the server

```
go run cmd/server/main.go
```