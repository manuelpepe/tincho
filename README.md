# Tincholi

A multiplayer card game for the whole family.

## Rules

The game is played in rounds, adding or subtracting points based on the sum of values of cards in their hands. The first player to cross 100 points ends the game.

Each player is dealt 4 face down cards. At the start of the game each player looks at the
values of two of their own cards and keeps them face down on the table. The fun of the game is trying to remember all cards you can and getting the lowest hand total before cutting. If you remember your hand you can call out your total when cutting to subtract extra points.

In turns, each player:

- Chooses to draw a card from either the Draw or Discard piles without showing it to other players.
- Decides to either:
    - Discarding the drawn card.
    - Discarding a card from their hand and keeping the drawed card in that position.
    - Trying to discard two cards from their hand at the same time, keeping only the drawed card (check how double discard works below)
    - Using the card's special effect (only 7s, 8s and 9s, check cards special effects below) and then discarding it.
    - Using the drawed card to cut, either calling out their count or not.
- If the player didn't cut, the turn passes to the next player.

When a player cut's, all hands are shown and points are calculated. Check below how cutting rules work in more details.

### Cards special effects

- 12 of Diamonds: This card is worth 0 points. Try not to discard it!
- All 7s: Peek the value of one card in your hand.
- All 8s: Peek the value of one card in an opponents hand.
- All 9s: Choose any two cards in the game and swap their places. This can be used to swap cards between any two players.

### Double Discard

When discarding, players have an option to discard two cards. If they are of the same value, both are discarded and the player keeps only the drawn card, *decreasing* the number of cards in their hand by a total of 1 card. However, if the cards are of different values, the player must keep both cards plus the drawn card, *increasing* the number of cards in their hand by a total of 1 card.

### Cutting rules

- The cutting player must have the lowest hand in the table, otherwise they instanly lose and must add their hand total + 20 points to their score
- If the cutting player has the lowest hand and decided to not call their total, their score this round is 0
- If the cutting player has the lowest hand and called their total right, their score is -10
- If the cutting player has the lowest hand and called their total wrong, their score is hand total + 10

All other players only add their hand totals to their score.

## Running the server

```
go run cmd/server/main.go
```