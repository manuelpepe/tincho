import "./types.js";

import { hide, show, moveNode } from "./utils.js";
import { SUITS, EFFECTS, EFFECT_SWAP, EFFECT_PEEK_OWN, EFFECT_PEEK_CARTA_AJENA, ACTION_DISCARD, ACTION_DISCARD_TWO } from "./constants.js";
import { queueAnimation, startProcessingAnimations } from "./animations.js";
import { setPlayerPeekedScreen, setStartGameScreen, setTurnScreen, setDrawScreen, setDiscardScreen, setStartRoundScreen, setCutScreen } from "./screens.js";

window.onload = function () {
    var suitKind = "standard"

    var CURRENT_ACTION = ACTION_DISCARD;

    /** @type {WebSocket} */
    var conn;

    /** @type {Object<string, {name: Element, checkmark: Element, score: Element, hand: Element, draw: Element, data: Player}>} */
    var PLAYERS = {};

    /** @type {string | null} */
    var THIS_PLAYER = null;

    /** @type {SwapBuffer | null} */
    var SWAP_BUFFER = null;

    /** @type {number | null} */
    var DISCARD_TWO_BUFFER = null;

    /** @type {boolean} */
    var FIRST_TURN = true;

    const NEXT_ROUND_TIMEOUT = 1000;
    const FIRST_PEEK_TIMEOUT = 1000;


    const roomid = /** @type {HTMLInputElement} */ (document.getElementById("room-id"));
    const username = /** @type {HTMLInputElement} */ (document.getElementById("username"));

    const roomTitle = document.getElementById("room-title");
    const formJoin = document.getElementById("room-join");
    const formNew = document.getElementById("room-new");
    const buttonStart = document.getElementById("btn-start");
    const buttonFirstPeek = document.getElementById("btn-first-peek");
    const buttonDraw = document.getElementById("btn-draw");
    const buttonDiscard = document.getElementById("btn-discard");
    const buttonDiscardTwo = document.getElementById("btn-discard-two");
    const buttonCancelDiscardTwo = document.getElementById("btn-cancel-discard-two");
    const buttonCut = document.getElementById("btn-cut");
    const buttonSwap = document.getElementById("btn-swap");
    const buttonPeekOwn = document.getElementById("btn-peek-own");
    const buttonPeekCartaAjena = document.getElementById("btn-peek-carta-ajena");

    const playerTemplate = /** @type {HTMLTemplateElement} */ (document.getElementById("player-template"))
    const playerList = document.getElementById("player-list");
    const playerContainer = document.getElementById("player-container");

    const gameContainer = document.getElementById("game");
    const endgameContainer = document.getElementById("endgame");

    const deckPile = document.getElementById("deck-pile");
    const deckDiscard = document.getElementById("deck-discard");


    /** @param {Card} card */
    function cardValue(card) {
        if (card.suit == "joker") {
            return SUITS[suitKind][card.suit]
        }
        return "" + card.value + SUITS[suitKind][card.suit]
    }

    /** 
     * @param {Element} sourcePile
     * @param {Element} target 
     */
    function drawCard(sourcePile, target) {
        const card = sourcePile.getElementsByClassName("card")[0];
        if (sourcePile == deckPile) {
            const cardClone = card.cloneNode(true);
            sourcePile.append(cardClone);
        }
        return moveNode(card, target);
    }

    /** 
     * @param {Player[]} new_players 
     */
    function setPlayers(new_players) {
        PLAYERS = {};
        playerList.innerHTML = "";
        playerContainer.innerHTML = "";
        for (let i = 0; i < new_players.length; i++) {
            if (new_players[i].id == THIS_PLAYER) {
                addPlayer(playerContainer, new_players[i]);
            } else {
                addPlayer(playerList, new_players[i]);
            }
        }
    }

    /** 
     * @param {HTMLElement} container
     * @param {Player} player 
     */
    function addPlayer(container, player) {
        let clone = /** @type {Element} */ (playerTemplate.content.cloneNode(true));
        let parts = clone.querySelectorAll(".player-datafield");
        drawHand(parts[1], player.id, player.cards_in_hand ?? 0, [], [])
        const playerData = {
            name: parts[0].getElementsByClassName("name")[0],
            checkmark: parts[0].getElementsByClassName("checkmark")[0],
            score: parts[0].getElementsByClassName("score")[0],
            hand: parts[1],
            draw: parts[2],
            data: player,
        };
        playerData.name.innerHTML = player.id;
        if (player.points) {
            playerData.score.innerHTML = "(" + player.points + ")";
        }
        PLAYERS[player.id] = playerData;
        container.appendChild(clone);
    }

    /** 
     * @param {string} player
     * @param {Card[]} cards 
     * @param {number[]} positions
     * @param {number} timeout
     */
    function showCards(player, cards, positions, timeout = 3000) {
        const playerHand = PLAYERS[player].hand;
        const playerData = PLAYERS[player].data;
        drawHand(playerHand, player, playerData.cards_in_hand, cards, positions)
        // TODO: Store timeout for skip button
        setTimeout(() => {
            drawHand(playerHand, player, playerData.cards_in_hand, [], [])
        }, timeout);
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
        PLAYERS[player].checkmark.innerHTML += " ✔";
    }

    function clearCheckmarks() {
        for (const player in PLAYERS) {
            PLAYERS[player].checkmark.innerHTML = "";
        }
    }

    /** 
     * @param {string} player
     * @param {string} source
     * @param {Card} card
     * @param {string} effect 
     */
    function showDraw(player, source, card, effect) {
        const playerDraw = PLAYERS[player].draw;
        switch (source) {
            case "pile":
                queueAnimation(
                    () => drawCard(deckPile, playerDraw),
                    () => {
                        playerDraw.innerHTML = "";
                        let text = card.suit ? "[" + cardValue(card) + "]" : "[ ]";
                        if (effect && effect != "none") {
                            text += " (Effect: " + EFFECTS[effect] + ")"
                        }
                        const textNode = document.createTextNode(text);
                        const node = createCardTemplate();
                        node.appendChild(textNode);
                        playerDraw.appendChild(node);
                    }
                );
                break;
            case "discard":
                queueAnimation(
                    () => drawCard(deckDiscard, playerDraw),
                    () => {
                        playerDraw.innerHTML = "";
                        let text = card.suit ? "[" + cardValue(card) + "]" : "[ ]";
                        if (effect && effect != "none") {
                            text += " (Effect: " + EFFECTS[effect] + ")"
                        }
                        const textNode = document.createTextNode(text);
                        const node = createCardTemplate();
                        node.appendChild(textNode);
                        playerDraw.appendChild(node);
                    }
                );
                break;
            default:
                console.error("unknown draw source: ", source)
                break;
        }
    }

    /** 
     * @param {string} player
     * @param {number} cardPosition
     * @param {Card} card
     */
    function showDiscard(player, cardPosition, card) {
        const playerHand = PLAYERS[player].hand;
        const playerDraw = PLAYERS[player].draw;
        const cardInHand = /** @type {HTMLElement} */ (playerHand.childNodes[cardPosition]);
        const tmpContainer = createCardTemplate();
        if (cardPosition >= 0) {
            queueAnimation(
                () => {
                    const drawnCard = /** @type {HTMLElement} */ (playerDraw.lastChild);
                    cardInHand.innerHTML = "[" + cardValue(card) + "]";
                    cardInHand.replaceWith(tmpContainer);
                    cardInHand.onclick = () => sendAction({
                        "type": "draw",
                        "data": { "source": "discard" },
                    });
                    tmpContainer.appendChild(cardInHand);
                    moveNode(drawnCard, tmpContainer);
                },
                () => {
                    const drawnCard = /** @type {HTMLElement} */ (tmpContainer.lastChild);
                    moveNode(cardInHand, deckDiscard)
                    while (deckDiscard.firstChild && deckDiscard.firstChild !== cardInHand) {
                        deckDiscard.removeChild(deckDiscard.firstChild);
                    }
                    tmpContainer.replaceWith(drawnCard);
                    drawnCard.innerHTML = "[ ]";
                    drawnCard.onclick = () => sendCurrentAction(player, cardPosition);
                }
            );
        } else {
            queueAnimation(
                () => {
                    const drawnCard = /** @type {HTMLElement} */ (playerDraw.lastChild);
                    drawnCard.innerHTML = "[" + cardValue(card) + "]"
                    drawnCard.onclick = () => sendAction({
                        "type": "draw",
                        "data": { "source": "discard" },
                    });
                    moveNode(drawnCard, deckDiscard)
                    while (deckDiscard.firstChild && deckDiscard.firstChild !== drawnCard) {
                        deckDiscard.removeChild(deckDiscard.firstChild);
                    }
                }
            );
        }
    }

    /**
     * @param {string} player
     * @param {number[]} cardPositions
     * @param {Card[]} cards
     */
    function showFailedDoubleDiscard(player, cardPositions, cards) {
        const playerHand = PLAYERS[player].hand;
        const playerDraw = PLAYERS[player].draw;
        for (let ix = 0; ix < cardPositions.length; ix++) {
            const cardInHand = /** @type {HTMLElement} */ (playerHand.childNodes[cardPositions[ix]]);
            cardInHand.innerHTML = "[" + cardValue(cards[ix]) + "]";
        }
        const tmpContainer = createCardTemplate();
        queueAnimation(
            () => {
                const drawnCard = /** @type {HTMLElement} */ (playerDraw.lastChild);
                playerHand.appendChild(tmpContainer);
                moveNode(drawnCard, tmpContainer)
            },
            () => {
                const drawnCard = /** @type {HTMLElement} */ (tmpContainer.lastChild);
                tmpContainer.replaceWith(drawnCard);
                drawnCard.onclick = () => sendCurrentAction(player, playerHand.childNodes.length - 1);
                // TODO: implement timeout with queueAnimation
                setTimeout(() => {
                    drawnCard.innerHTML = "[ ]";
                    for (let ix = 0; ix < cardPositions.length; ix++) {
                        const cardInHand = /** @type {HTMLElement} */ (playerHand.childNodes[cardPositions[ix]]);
                        cardInHand.innerHTML = "[ ]";
                    }
                }, 1000);
            }
        );
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
        const playerOneHand = PLAYERS[players[0]].hand;
        const playerOneCard = /** @type {HTMLElement} */ (playerOneHand.childNodes[cardPositions[0]]);
        const playerTwoHand = PLAYERS[players[1]].hand;
        const playerTwoCard = /** @type {HTMLElement} */ (playerTwoHand.childNodes[cardPositions[1]]);
        const tmpContainerOne = createCardTemplate();
        const tmpContainerTwo = createCardTemplate();
        queueAnimation(
            () => {
                playerOneCard.replaceWith(tmpContainerOne);
                tmpContainerOne.appendChild(playerOneCard);
                playerTwoCard.replaceWith(tmpContainerTwo)
                tmpContainerTwo.appendChild(playerTwoCard);
                moveNode(playerOneCard, tmpContainerTwo, 2000);
            },
            () => {
                moveNode(playerTwoCard, tmpContainerOne, 2000);
            },
            () => {
                tmpContainerOne.replaceWith(playerTwoCard);
                tmpContainerTwo.replaceWith(playerOneCard);
                playerOneCard.onclick = () => sendCurrentAction(players[0], cardPositions[0]);
                playerTwoCard.onclick = () => sendCurrentAction(players[1], cardPositions[1]);
            }
        );
    }

    /** 
     * @param {string} player
     * @param {boolean} withCount
     * @param {number} declared 
     */
    function showCut(player, withCount, declared, hands) {
        // TODO: Show player, withcount and declared
        for (const [ix, [player, data]] of Object.entries(PLAYERS).entries()) {
            const positions = [...Array(hands[ix].length).keys()];
            showCards(player, hands[ix], positions, NEXT_ROUND_TIMEOUT);
        }
    }

    /** @param {{playerID: string, score: number}[][]} scores */
    function showEndGame(scores) {
        console.log(scores)
        hide(gameContainer);
        const tbl = document.createElement("table");
        const tblBody = document.createElement("tbody");
        for (const [ix, round] of Object.entries(scores)) {
            const row = document.createElement("tr");
            const roundCell = document.createElement("td");
            const roundNum = ix + 1;
            roundCell.appendChild(document.createTextNode("Round " + roundNum));
            row.appendChild(roundCell);
            for (const score of round) {
                const cell1 = document.createElement("td");
                const cell2 = document.createElement("td");
                cell1.appendChild(document.createTextNode(score["playerID"]));
                cell2.appendChild(document.createTextNode("" + score["score"]));
                row.appendChild(cell1);
                row.appendChild(cell2);
            }
            tblBody.appendChild(row);

        }
        tbl.appendChild(tblBody);
        endgameContainer.appendChild(tbl);
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
        console.log("Received message:", data)
        switch (data.type) {
            case "players_changed":
                setPlayers(msgData.players)
                break;
            case "game_start":
                FIRST_TURN = true;
                setStartGameScreen();
                setPlayers(msgData.players)
                break;
            case "player_peeked":
                if (msgData.player == THIS_PLAYER) {
                    setPlayerPeekedScreen()
                    showCards(msgData.player, msgData.cards, [0, 1])
                }
                markReady(msgData.player)
                break;
            case "turn":
                if (FIRST_TURN) {
                    clearCheckmarks();
                    FIRST_TURN = false;
                }
                setTurnScreen(msgData.player == THIS_PLAYER);
                break;
            case "draw":
                showDraw(msgData.player, msgData.source, msgData.card, msgData.effect);
                setDrawScreen(msgData.player == THIS_PLAYER, msgData.effect)
                break;
            case "discard":
                for (let ix = 0; ix < msgData.card.length; ix++) {
                    showDiscard(msgData.player, msgData.cardPosition[ix], msgData.card[ix])
                }
                setDiscardScreen();
                break;
            case "failed_double_discard":
                showFailedDoubleDiscard(msgData.player, msgData.cardPositions, msgData.cards);
                break;
            case "effect_peek":
                showPeek(msgData.player, msgData.cardPosition, msgData.card)
                break;
            case "effect_swap":
                showSwap(msgData.players, msgData.cardPositions)
                break;
            case "cut":
                setPlayers(msgData.players)
                showCut(msgData.player, msgData.withCount, msgData.declared, msgData.hands);
                setCutScreen();
                break;
            case "start_next_round":
                // TODO: Refactor to show next round button
                setTimeout(() => {
                    FIRST_TURN = true;
                    setStartRoundScreen();
                    setPlayers(msgData.players)
                }, NEXT_ROUND_TIMEOUT);
                break;
            case "end_game":
                showEndGame(msgData.scores);
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
        conn = new WebSocket("ws://" + location.host + "/join?room=" + roomid.value + "&player=" + username.value);
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
        await fetch("http://" + location.host + "/new")
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
        "data": { "withCount": true, "declared": 3 }, // TODO: Implement cut configuration by player
    });

    buttonDiscard.onclick = () => sendDiscard(-1);
    buttonDiscardTwo.onclick = () => {
        setAction(ACTION_DISCARD_TWO);
        hide(buttonDiscardTwo);
        show(buttonCancelDiscardTwo);
        DISCARD_TWO_BUFFER = null;
    };
    buttonCancelDiscardTwo.onclick = () => {
        setAction(ACTION_DISCARD_TWO);
        show(buttonDiscardTwo);
        hide(buttonCancelDiscardTwo);
        DISCARD_TWO_BUFFER = null;
    }

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
            case ACTION_DISCARD_TWO:
                if (player != THIS_PLAYER) {
                    console.log("can't discard another player's card");
                    return;
                }
                if (DISCARD_TWO_BUFFER == null) {
                    DISCARD_TWO_BUFFER = cardPos;
                    console.log("Set discard two buffer to: ", DISCARD_TWO_BUFFER);
                    return;
                }
                sendDiscard(DISCARD_TWO_BUFFER, cardPos);
                DISCARD_TWO_BUFFER = null;
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

    /** 
     * @param {number} ix 
     * @param {number} ix2 
     * */
    function sendDiscard(ix, ix2 = null) {
        sendAction({
            "type": "discard",
            "data": { "cardPosition": ix, "cardPosition2": ix2 },
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

    startProcessingAnimations();
};