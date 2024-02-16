### Features

- [x] Implement start game peek (2 cards from own hand each)
- [x] Separate broadcasting logic from game logic
    - [x] Remove playerID from methods and use current turn
    - [x] Make all attrs private in state component
    - [x] Refactor room to use the state component
- [x] Keep track of scores per round and send table every round
- [x] Show scores in UI
- [x] Rooms should timeout after a while
- [ ] Implement turn time limit
- [ ] Implement reconnection
    - Option 1: Verify (IP, Username, RoomID)
    - Option 2:
        - Generate token on join
        - Send token to user
        - Client stores token in local storage along with username
        - Allow client to reconnect if using token
    - Send necessary state on reconnection
    - Prevent duplicated players from joining game (prevent dup join without token).
- [ ] Make room parametrizable:
    - Options:
        - [ ] Extended deck option (add some cards to allow for more players)
        - [ ] Chaos (Adds two 9 cards for more swaps)
        - [ ] Make private (set password for room)
    - [ ] UI for room creation
- [ ] Roomlist in UI with join buttons and private status
- [ ] Add styles to UI
- [ ] Only room leader can start room
    - [ ] Transfer leadership to other players 
- [ ] Bots:
    - Room should process messages with channels instead of interacting directly with the websocket
    - Handler should manage socket and use channels to communicate with the room
    - Bot variations:
        1. [x] Easy: 
            - Performs actions completly randomly
                - 50% draw from draw pile, 50% draw from discard pile
                - 50% discard drawn 45% discard from hand 5% double discard
            - Doesn't keep track of any card
            - Doesn't activate any effect (peek, swap)
            - Cuts randomly without declaring (5% chance every turn)
        2. Medium: 
            - Keeps track of cards in hand
            - Discards unknown cards first
            - Always discard the highest card (20% chance of mistake)
            - Doesn't double discard
            - Doesn't swap or peek opponents cards cards
            - Does peek own cards
            - 75% chance of cutting when <= 10 points (5% chance of mistake)
            - Never declares hand
        3. Hard:
            - Keeps track of cards in hand
            - Discards unknown cards first
            - Always discard the highest card (0% chance of mistake)
            - Only double discards if it peeks a repeated card
            - Does swap oppenents cards for chaos
            - Does peek own cards
            - Only cuts with =< 5 points (0% chance of mistake)
            - Always declares hand
        4. Expert:
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

- [ ] When peeking, send peek position to all players so they know which card has been peeked (but not the value)
- [ ] When swapping cards, an intermediate container should be used to pass them around (improves animation)
- [ ] UI error when peeking 5th opponent card after failed double discard
- [ ] Prevent discarding drawed card if it was drawed from the discard pile