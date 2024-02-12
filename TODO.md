### Features

- [x] Implement start game peek (2 cards from own hand each)
- [x] Separate broadcasting logic from game logic
    - [x] Remove playerID from methods and use current turn
    - [x] Make all attrs private in state component
    - [x] Refactor room to use the state component
- [x] Keep track of scores per round and send table every round
- [ ] Show scores in UI
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


### Fixes

- [ ] When peeking, send peek position to all players so they know which card has been peeked (but not the value)
- [ ] When swapping cards, an intermediate container should be used to pass them around (improves animation)