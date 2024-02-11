window.onload = function () {
    /** @typedef {{suit: string, value: number}} Card */
    /** @typedef {{id: string, points: number, pending_first_peek: boolean, cards_in_hand: number}} Player */
    /** @typedef {{player: string, cardPosition: number}} SwapBuffer */

    const SUITS = {
        "spanish": {
            "clubs": "B",
            "hearts": "C",
            "diamonds": "O",
            "spades": "E",
            "joker": "J",
        },
        "standard": {
            "clubs": "♧",
            "hearts": "♥",
            "diamonds": "♢",
            "spades": "♤",
            "joker": "J",
        }
    }

    const EFFECT_SWAP = "swap_card"
    const EFFECT_PEEK_OWN = "peek_own"
    const EFFECT_PEEK_CARTA_AJENA = "peek_carta_ajena"
    const ACTION_DISCARD = "discard"

    var CURRENT_ACTION = ACTION_DISCARD;

    const EFFECTS = {
        [EFFECT_SWAP]: "Swap 2 cards",
        [EFFECT_PEEK_OWN]: "Peek card from your hand",
        [EFFECT_PEEK_CARTA_AJENA]: "Peek card from other player"
    }
    var suitKind = "standard"

    /** @param {Card} card */
    function cardValue(card) {
        if (card.suit == "joker") {
            return SUITS[suitKind][card.suit]
        }
        return "" + card.value + SUITS[suitKind][card.suit]
    }

    /** @type {WebSocket} */
    var conn;

    /** @type {Object<string, {hand: Element, draw: Element, data: Player}>} */
    var PLAYERS = {};

    /** @type {string | null} */
    var THIS_PLAYER = null;

    /** @type {SwapBuffer | null} */
    var SWAP_BUFFER = null;

    const roomid = /** @type {HTMLInputElement} */ (document.getElementById("room-id"));
    const username = /** @type {HTMLInputElement} */ (document.getElementById("username"));

    const roomTitle = document.getElementById("room-title");
    const formJoin = document.getElementById("room-join");
    const formNew = document.getElementById("room-new");
    const buttonStart = document.getElementById("btn-start");
    const buttonFirstPeek = document.getElementById("btn-first-peek");
    const buttonDraw = document.getElementById("btn-draw");
    const buttonDiscard = document.getElementById("btn-discard");
    const buttonCut = document.getElementById("btn-cut");
    const buttonSwap = document.getElementById("btn-swap");
    const buttonPeekOwn = document.getElementById("btn-peek-own");
    const buttonPeekCartaAjena = document.getElementById("btn-peek-carta-ajena");

    const playerTemplate = /** @type {HTMLTemplateElement} */ (document.getElementById("player-template"))
    const playerList = document.getElementById("player-list");

    const deckPile = document.getElementById("deck-pile");
    const deckDiscard = document.getElementById("deck-discard");

    /** @param {HTMLElement} node */
    function hide(node) {
        node.style.display = "none";
    }

    /** @param {HTMLElement} node */
    function show(node) {
        node.style.display = "block";
    }

    /** 
     * @param {Element} node
     * @param {Element} target 
     * @param {number | null} duration
     */
    function moveNode(node, target, duration = 1000) {
        if (duration == null || duration == undefined) {
            duration = 1000;
        }
        const { left: x0, top: y0 } = node.getBoundingClientRect();
        target.append(node);
        const { left: x1, top: y1 } = node.getBoundingClientRect();

        const dx = x0 - x1;
        const dy = y0 - y1;

        if (dx === 0 && dy === 0) {
            return;
        }

        const transformFrom = `translate3d(${dx}px, ${dy}px, 0)`;
        const transformTo = `translate3d(0, 0, 0)`;

        const animation = node.animate([
            { transform: transformFrom },
            { transform: transformTo },
        ], {
            duration: duration,
            easing: 'linear',
        });
        return animation
    }

    /** 
     * @param {Element} sourcePile
     * @param {Element} target 
     */
    function drawCard(sourcePile, target) {
        const card = sourcePile.getElementsByClassName("card")[0];
        const cardClone = card.cloneNode(true);
        sourcePile.append(cardClone);
        return moveNode(card, target);
    }

    /** 
     * @param {Player[]} new_players 
     */
    function setPlayers(new_players) {
        console.log("players", new_players)
        PLAYERS = {};
        playerList.innerHTML = "";
        for (let i = 0; i < new_players.length; i++) {
            addPlayer(new_players[i]);
        }
    }

    /** 
     * @param {Player} player 
     */
    function addPlayer(player) {
        let clone = /** @type {Element} */ (playerTemplate.content.cloneNode(true));
        let parts = clone.querySelectorAll(".player-datafield");
        parts[0].innerHTML = player.id;
        drawHand(parts[1], player.id, player.cards_in_hand ?? 0, [], [])
        PLAYERS[player["id"]] = {
            "hand": parts[1],
            "draw": parts[2],
            "data": player,
        };
        playerList.appendChild(clone);
    }

    /** 
     * @param {string} player
     * @param {Card[]} cards 
     * @param {number[]} positions
     */
    function showCards(player, cards, positions) {
        const playerHand = PLAYERS[player]["hand"];
        const playerData = PLAYERS[player]["data"];
        drawHand(playerHand, player, playerData.cards_in_hand, cards, positions)
        // TODO: Store timeout for skip button
        setTimeout(() => {
            drawHand(playerHand, player, playerData.cards_in_hand, [], [])
        }, 3000);
    }

    /** 
     * @param {Element} container
     * @param {string} player
     * @param {number} cardsInHand
     * @param {Card[]} cards
     * @param {number[]} positions 
     */
    function drawHand(container, player, cardsInHand, cards, positions) {
        while (container.firstChild) { container.removeChild(container.lastChild) }
        let cardix = 0;
        for (let i = 0; i < cardsInHand; i++) {
            let text;
            if (positions && positions[cardix] == i) {
                text = document.createTextNode("[" + cardValue(cards[cardix]) + "]");
                cardix = cardix + 1
            } else {
                text = document.createTextNode("[ ]");
            }
            const node = createCardTemplate();
            node.onclick = () => sendCurrentAction(player, i)
            node.appendChild(text);
            container.appendChild(node);
        }
    }

    /** @param {string} player */
    function markReady(player) {
        // TODO: implement checkmark
        console.log(`player ready: ${player}`)
    }

    /** 
     * @param {string} player
     * @param {string} source
     * @param {Card} card
     * @param {string} effect 
     */
    function showDraw(player, source, card, effect) {
        // TODO: Draw from discard pile
        const playerDraw = PLAYERS[player]["draw"];
        const animation = drawCard(deckPile, playerDraw);
        animation.addEventListener("finish", () => {
            playerDraw.innerHTML = "";
            let text = card.suit ? "[" + cardValue(card) + "]" : "[ ]";
            if (effect && effect != "none") {
                text += " (Effect: " + EFFECTS[effect] + ")"
            }
            const textNode = document.createTextNode(text);
            const node = createCardTemplate();
            node.appendChild(textNode);
            playerDraw.appendChild(node);
        });
    }

    /** 
     * @param {string} player
     * @param {number} cardPosition
     * @param {Card} card
     */
    function showDiscard(player, cardPosition, card) {
        const playerHand = PLAYERS[player]["hand"];
        const playerDraw = PLAYERS[player]["draw"];
        const drawnCard = /** @type {HTMLElement} */ (playerDraw.lastChild);
        if (cardPosition >= 0) {
            const tmpContainer = createCardTemplate();
            const cardInHand = /** @type {HTMLElement} */ (playerHand.childNodes[cardPosition]);
            cardInHand.innerHTML = "[" + cardValue(card) + "]";
            cardInHand.replaceWith(tmpContainer);
            tmpContainer.appendChild(cardInHand);
            const animation = moveNode(drawnCard, tmpContainer);
            animation.addEventListener("finish", () => {
                moveNode(cardInHand, deckDiscard)
                tmpContainer.replaceWith(drawnCard);
                drawnCard.innerHTML = "[ ]";
                drawnCard.onclick = () => sendCurrentAction(player, cardPosition);
            });
        } else {
            const animation = moveNode(drawnCard, deckDiscard);
            animation.addEventListener("finish", () => {
                drawnCard.innerHTML = "[" + cardValue(card) + "]"
            })
        }
    }

    /** 
     * @param {string} player
     * @param {number} cardPosition
     * @param {Card} card 
     */
    function showPeek(player, cardPosition, card) {
        showCards(player, [card], [cardPosition])
    }

    /** 
     * @param {string[]} players
     * @param {number[]} cardPositions 
     */
    function showSwap(players, cardPositions) {
        const playerOneHand = PLAYERS[players[0]]["hand"];
        const playerOneCard = /** @type {HTMLElement} */ (playerOneHand.childNodes[cardPositions[0]]);
        const playerTwoHand = PLAYERS[players[1]]["hand"];
        const playerTwoCard = /** @type {HTMLElement} */ (playerTwoHand.childNodes[cardPositions[1]]);

        const tmpContainerOne = createCardTemplate();
        const tmpContainerTwo = createCardTemplate();
        playerOneCard.replaceWith(tmpContainerOne);
        tmpContainerOne.appendChild(playerOneCard);
        playerTwoCard.replaceWith(tmpContainerTwo)
        tmpContainerTwo.appendChild(playerTwoCard);

        const anim = moveNode(playerOneCard, tmpContainerTwo, 2000);
        anim.addEventListener("finish", () => {
            const anim = moveNode(playerTwoCard, tmpContainerOne, 2000);
            anim.addEventListener("finish", () => {
                tmpContainerOne.replaceWith(playerTwoCard);
                tmpContainerTwo.replaceWith(playerOneCard);
                playerOneCard.onclick = () => sendCurrentAction(players[0], cardPositions[0]);
                playerTwoCard.onclick = () => sendCurrentAction(players[1], cardPositions[1]);
            })
        });

    }

    /** 
     * @param {string} player
     * @param {boolean} withCount
     * @param {number} declared 
     */
    // eslint-disable-next-line no-unused-vars
    function showCut(player, withCount, declared) { /* TODO */ }

    /** @param {string} winner */
    // eslint-disable-next-line no-unused-vars
    function showEndGame(winner) { /* TODO */ }

    /** 
     *  @param {string} effect 
     *  @returns {HTMLElement | null}
    */
    function getEffectButton(effect) {
        switch (effect) {
            case EFFECT_SWAP:
                return buttonSwap
            case EFFECT_PEEK_OWN:
                return buttonPeekOwn
            case EFFECT_PEEK_CARTA_AJENA:
                return buttonPeekCartaAjena
            case "none":
            case "":
                break;
            default:
                console.log("Unkown effect:", effect)
        }
        return null
    }

    /** @param {string} effect */
    function showEffectButton(effect) {
        let btn = getEffectButton(effect)
        if (btn != null) {
            show(btn)
        }
    }

    function hideEffectButtons() {
        hide(buttonSwap);
        hide(buttonPeekOwn);
        hide(buttonPeekCartaAjena);
    }

    /** @param {string} action */
    function setAction(action) {
        // TODO: check if action is valid
        console.log("Setting action to: ", action)
        CURRENT_ACTION = action
    }

    /** @param {MessageEvent<any>} event} */
    function processWSMessage(event) {
        const data = JSON.parse(event.data)
        const msgData = data.data;
        let fn;
        console.log("Received message:", data)
        switch (data.type) {
            case "players_changed":
                setPlayers(msgData.players)
                break;
            case "game_start":
                hide(buttonStart)
                show(buttonFirstPeek)
                show(deckPile)
                show(deckDiscard)
                setPlayers(msgData.players)
                break;
            case "player_peeked":
                if (msgData.player == THIS_PLAYER) {
                    hide(buttonFirstPeek)
                    showCards(msgData.player, msgData.cards, [0, 1])
                }
                markReady(msgData.player)
                break;
            case "turn":
                fn = msgData.player == username.value ? show : hide;
                fn(buttonDraw)
                fn(buttonDiscard)
                fn(buttonCut)
                break;
            case "draw":
                showDraw(msgData.player, msgData.source, msgData.card, msgData.effect)
                showEffectButton(msgData.effect)
                break;
            case "discard":
                if (msgData.card.length == 1) {
                    showDiscard(msgData.player, msgData.cardPosition[0], msgData.card[0])
                }
                // TODO: Handle double discard
                hideEffectButtons();
                break;
            case "effect_peek":
                showPeek(msgData.player, msgData.cardPosition, msgData.card)
                break;
            case "effect_swap":
                showSwap(msgData.players, msgData.cardPositions)
                // TODO: Animate swap cards and discard drawn
                break;
            case "cut":
                showCut(msgData.player, msgData.withCount, msgData.declared)
                setPlayers(msgData.players)
                // TODO: Animate show scores
                break;
            case "end_game":
                showEndGame(msgData.winner)
                break;
            default:
                console.error("Unknown message type", data.type, msgData)
                break;
        }

    }

    function connectToRoom() {
        if (!roomid.value) {
            return false;
        }
        conn = new WebSocket("ws://localhost:5555/join?room=" + roomid.value + "&player=" + username.value);
        conn.onclose = () => console.log("connection closed");
        conn.onmessage = processWSMessage;
        hide(formNew);
        hide(formJoin);
        roomTitle.innerHTML = "Room " + roomid.value;
        show(roomTitle);
        show(buttonStart);
        console.log("connected to room " + roomid.value);
        THIS_PLAYER = username.value;
        return false;
    }

    formNew.onsubmit = async (evt) => {
        evt.preventDefault();
        await fetch("http://localhost:5555/new")
            .then(response => response.text())
            .then(data => roomid.value = data);
        connectToRoom();
    };


    formJoin.onsubmit = (evt) => {
        evt.preventDefault();
        connectToRoom();
    }

    buttonStart.onclick = () => sendAction({
        "type": "start",
        "data": {},
    });

    buttonFirstPeek.onclick = () => sendAction({
        "type": "first_peek",
        "data": { "positions": [0, 1] },
    });

    buttonDraw.onclick = () => sendAction({
        "type": "draw",
        "data": { "source": "pile" },
    });

    buttonCut.onclick = () => sendAction({
        "type": "cut",
        "data": { "withCount": true, "declared": 3 },
    });

    buttonDiscard.onclick = () => sendDiscard(-1);

    buttonSwap.onclick = () => setAction(EFFECT_SWAP);
    buttonPeekOwn.onclick = () => setAction(EFFECT_PEEK_OWN);
    buttonPeekCartaAjena.onclick = () => setAction(EFFECT_PEEK_CARTA_AJENA);


    /** 
     * @param {string} player
     * @param {number} cardPos 
    */
    function sendCurrentAction(player, cardPos) {
        console.log("Handling current action: ", CURRENT_ACTION);
        switch (CURRENT_ACTION) {
            case ACTION_DISCARD:
                if (player != THIS_PLAYER) {
                    console.log("can't discard another player's card");
                    return;
                }
                sendDiscard(cardPos);
                break;
            case EFFECT_SWAP:
                if (SWAP_BUFFER == null) {
                    SWAP_BUFFER = { player: player, cardPosition: cardPos };
                    console.log("Set swap buffer to: ", SWAP_BUFFER);
                    return;
                }
                sendAction({
                    "type": "effect_swap_card",
                    "data": {
                        "cardPositions": [SWAP_BUFFER.cardPosition, cardPos],
                        "players": [SWAP_BUFFER.player, player],
                    }
                });
                SWAP_BUFFER = null;
                break;
            case EFFECT_PEEK_OWN:
                if (player != THIS_PLAYER) {
                    console.log("peek a card from your own hand");
                    return;
                }
                sendAction({
                    "type": "effect_peek_own",
                    "data": {
                        "cardPosition": cardPos,
                    },
                });
                break;
            case EFFECT_PEEK_CARTA_AJENA:
                if (player == THIS_PLAYER) {
                    console.log("peek a card from another player");
                    return;
                }
                sendAction({
                    "type": "effect_peek_carta_ajena",
                    "data": {
                        "cardPosition": cardPos,
                        "player": player,
                    }
                });
                break;
        }
        setAction(ACTION_DISCARD);
    }

    /** @param {number} ix */
    function sendDiscard(ix) {
        sendAction({
            "type": "discard",
            "data": { "cardPosition": ix },
        });
    }

    /** @param {Object} data */
    function sendAction(data) {
        if (!conn) {
            console.error("WS not connected");
            return false;
        }
        console.log("Sent data:", data)
        conn.send(JSON.stringify(data));
    }

    /** @returns {HTMLElement} */
    function createCardTemplate() {
        const card = document.createElement("div");
        card.className = "card";
        return card;
    }
};