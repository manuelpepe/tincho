## Pending

### Features

- [ ] Should probably replace GameStart event with StartNextRound 
- [ ] Changelog in UI
- [ ] Rejoin to before-start, first-peek and cut screens.
- [ ] Improved error messages
- [ ] Save games in disk for analysis
- [ ] [FRONT] Display withCount and declared info on cut screen
- [ ] [BACK] Turn time limit (probably should draw and discard drawed card)
- [ ] [BACK+FRONT] Roomlist in UI with join buttons and private status
    - [ ] [BACK] Check room listing, add room capacity
    - [ ] [FRONT] "Search games" option in menu, list component
- [ ] [FRONT] Add styles to UI (this will never be finished)
- [ ] [BACK] Bots:
    1. [ ] Expert:
        - Keeps track of cards in hand
        - Discards unknown cards first
        - Always discard the highest card (0% chance of mistake)
        - Actively tries to double discard
            - Stores drawn card if one of similar value is in hand and one of a higher value can be discarded
            - Only double discards for a card of less total value
        - Keeps track of oponent hands and swaps for advantage (todo: define advantage)
        - Does peek own and unkown opponents cards
        - Only cuts with =< 5 points (0% chance of mistake)
        - Doesn't cut if it knows an opponent has less points
        - Always declares hand

### Fixes

* [ ] [PERF-SIM] Most of the time in simulations is spent in json.Unmarshall calls in the bot handler
* [ ] [FRONT] On failed double discard with empty discard pile, animate a card going from draw to discard instead of magically appearing. Don't make the same animation if the discard pile isn't emtpy because the card showing would already be in the discard pile.
- [ ] [CANT-REPRODUCE] On mobile, sometimes the click handler doesn't work after swapping cards.
- [ ] [CANT-REPRODUCE] On mobile, an eye appears instead of card value when peeking own card.
- [ ] [FRONT] When swapping cards, an intermediate container should be used to pass them around (improves animation)
- [ ] [BACK] Prevent discarding drawed card if it was drawed from the discard pile



## Completed

### Features

- [x] Implement start game peek (2 cards from own hand each)
- [x] Separate broadcasting logic from game logic
    - [x] Remove playerID from methods and use current turn
    - [x] Make all attrs private in state component
    - [x] Refactor room to use the state component
- [x] Keep track of scores per round and send table every round
- [x] Show scores in UI
- [x] Rooms should timeout after a while
- [x] Only room leader can start room
- [-] Bots:
    - [x] Room should process messages with channels instead of interacting directly with the websocket
    - [x] Handler should manage socket and use channels to communicate with the room
    - Bot variations:
        1. [x] Easy: 
            - Performs actions completly randomly
                - 50% draw from draw pile, 50% draw from discard pile
                - 50% discard drawn 45% discard from hand 5% double discard
            - Doesn't keep track of any card
            - Doesn't activate any effect (peek, swap)
            - Cuts randomly without declaring (5% chance every turn)
        2. [x] Medium: 
            - Keeps track of cards in hand
            - Discards unknown cards first
            - Always discard the highest card (20% chance of mistake)
            - Doesn't double discard
            - Doesn't swap or peek opponents cards cards
            - Does peek own cards
            - 75% chance of cutting when <= 10 points (5% chance of mistake)
            - Never declares hand
        3. [x] Hard:
            - Keeps track of cards in hand
            - Discards unknown cards first
            - Always discard the highest card (0% chance of mistake)
            - Only double discards if it peeks a repeated card
            - Does swap oppenents cards for chaos
            - Does peek own cards
            - Only cuts with <= 6 points (0% chance of mistake)
            - Always declares hand
- [x] Make room parametrizable:
    - Options:
        - [x] Extended deck option (add some cards to allow for more players)
        - [x] Chaos (Adds two 9 cards for more swaps)
        - [x] Make private (set password for room)
        - [x] Max players
    - [x] UI for room creation
- [x] Reconnection
    - Generate token on join
    - Send token to user in cookie
    - Allow client to reconnect if using token
    - Send necessary state on reconnection
    - Prevent duplicated players from joining game (prevent dup join without token).
- [x] Configurable animation speed
- [x] Display more info on cut
- [x] Skip button for cut screen and first peek
- [x] With round history also store: cut info, all hands
- [x] Automatic rejoin
- [x] Draw card at the start of round.
- [x] Add indicator of current turn
- [x] Add a `piles shuffled` update
- [x] Add a `piles shuffled` animation
- [x] Rules in UI
- [x] Prevent using effect if card was drawn from discard pile

### Fixes

- [x] When peeking, send peek position to all players so they know which card has been peeked (but not the value)
- [x] UI error when peeking 5th opponent card after failed double discard
- [x] Points are not being correctly calculated
- [x] Bots can draw from empty discard deck and crash
- [x] Refactor `queueAnimation` to `queueAction` and queue actions as they come in through the socket. queuing only animations results in ui bugs.
- [x] [FRONT] After cutting, timeout runs for every player, hiding their cards in order instead of all at once, and creating long wait times
- [x] End game screen is all wrong
- [x] On fail double discard, when draw was from discard pile:
