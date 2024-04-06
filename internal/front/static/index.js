import "./types.js";

import { hide, show, moveNode, createCardTemplate } from "./utils.js";
import { SUITS, EFFECTS, EFFECT_SWAP, EFFECT_PEEK_OWN, EFFECT_PEEK_CARTA_AJENA, ACTION_DISCARD, ACTION_DISCARD_TWO } from "./constants.js";
import { queueActions, queueActionInstantly, startProcessingActions } from "./actions.js";
import { setPlayerPeekedScreen, setStartGameScreen, setTurnScreen, setDrawScreen, setDiscardScreen, setStartRoundScreen, setCutScreen } from "./screens.js";
import { PEEK_TIMEOUT, NEXT_ROUND_TIMEOUT, SWAP_DURATION } from './configs.js';
import { getWaiter } from "./utils.js";

window.onload = function () {
    var suitKind = "standard"

    var CURRENT_ACTION = ACTION_DISCARD;

    /** @type {WebSocket} */
    var conn;

    /** @type {Object<string, {name: Element, checkmark: Element, score: Element, hand: Element, draw: Element, data: Player}>} */
    var PLAYERS = {};

    /** @type {string | null} */
    var THIS_PLAYER = null;

    /** @type {string | null} */
    var THIS_ROOM = null;

    /** @type {SwapBuffer | null} */
    var SWAP_BUFFER = null;

    /** @type {number | null} */
    var DISCARD_TWO_BUFFER = null;

    /** @type {boolean} */
    var FIRST_TURN = true;

    const joinMenuRoomID = /** @type {HTMLInputElement} */ (document.getElementById("join-room-id"));
    const joinMenuUsername = /** @type {HTMLInputElement} */ (document.getElementById("join-username"));
    const joinMenuPassword = /** @type {HTMLInputElement} */ (document.getElementById("join-password"));
    const createMenuUsername = /** @type {HTMLInputElement} */ (document.getElementById("create-username"));
    const createMenuMaxPlayers = /** @type {HTMLInputElement} */ (document.getElementById("max-players"));
    const createMenuPassword = /** @type {HTMLInputElement} */ (document.getElementById("password"));
    const createMenuUseExtendedDeck = /** @type {HTMLInputElement} */ (document.getElementById("use-extended-deck"));
    const createMenuUseChaosDeck = /** @type {HTMLInputElement} */ (document.getElementById("use-chaos-deck"));

    const menuContainer = document.getElementById("menu-container");
    const mainMenu = document.getElementById("main-menu");
    const buttonShowCreateMenu = document.getElementById("btn-show-create-menu");
    const buttonShowJoinMenu = document.getElementById("btn-show-join-menu");
    const menuJoin = document.getElementById("join-menu");
    const menuCreate = document.getElementById("create-menu");

    const buttonJoinRooom = document.getElementById("room-join");
    const buttonNewRoom = document.getElementById("room-new");

    const selectBotDiff = /** @type {HTMLSelectElement} */ (document.getElementById("bot-diff-select"));
    const buttonAddBot = document.getElementById("btn-add-bot");

    const buttonStart = document.getElementById("btn-start");
    const buttonFirstPeek = document.getElementById("btn-first-peek");
    const buttonDraw = document.getElementById("btn-draw");
    const buttonDiscard = document.getElementById("btn-discard");
    const buttonDiscardTwo = document.getElementById("btn-discard-two");
    const buttonCancelDiscardTwo = document.getElementById("btn-cancel-discard-two");
    const buttonSwap = document.getElementById("btn-swap");
    const buttonPeekOwn = document.getElementById("btn-peek-own");
    const buttonPeekCartaAjena = document.getElementById("btn-peek-carta-ajena");
    const buttonContinue = document.getElementById("btn-continue");

    const buttonCut = document.getElementById("btn-cut");
    const inputCutDeclare = /** @type {HTMLInputElement} */ (document.getElementById("input-cut-declare"));
    const inputCutDeclared = /** @type {HTMLInputElement} */ (document.getElementById("input-cut-declared"));
    const cutInfoDialog = document.getElementById("cut-info-dialog");

    const playerTemplate = /** @type {HTMLTemplateElement} */ (document.getElementById("player-template"))
    const playerList = document.getElementById("player-list");
    const playerContainer = document.getElementById("player-container");

    const gameContainer = document.getElementById("game");
    const endgameContainer = document.getElementById("endgame");

    const deckPile = document.getElementById("deck-pile");
    const deckDiscard = document.getElementById("deck-discard");

    const errorContainer = document.getElementById("error-container");

    let waiter = getWaiter();

    async function waitUserInput() {
        buttonContinue.onclick = () => waiter.notify();
        show(buttonContinue);
        await waiter.wait();
        hide(buttonContinue);
    }

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
    async function drawCard(sourcePile, target) {
        const card = sourcePile.getElementsByClassName("card")[0];
        if (sourcePile == deckPile) {
            const cardClone = card.cloneNode(true);
            sourcePile.append(cardClone);
        }
        return await moveNode(card, target);
    }

    /** @param {Card} card */
    function setLastDiscarded(card) {
        const node = createCardTemplate();
        node.appendChild(document.createTextNode("[" + cardValue(card) + "]"));
        node.onclick = () => sendAction({
            "type": "draw",
            "data": { "source": "discard" },
        });
        deckDiscard.appendChild(node);
        clearDiscardDeckExcept(node);
    }

    /**
     * card can be null when showing another player's draw
     * @param {string} player 
     * @param {Card | null} card 
     */
    function setCardInDraw(player, card) {
        const playerDraw = PLAYERS[player].draw;
        const node = createCardTemplate();
        let valueText = document.createTextNode(card ? "[" + cardValue(card) + "]" : "[ ]");
        node.appendChild(valueText);
        playerDraw.appendChild(node);
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
     * @param {boolean} withCount
     * @param {number} declared
     */
    function setCutInfo(player, withCount, declared) {
        show(cutInfoDialog);
        cutInfoDialog.innerHTML = `Player ${player} cut ${withCount ? `declaring ${declared}` : "without declaring"}`;
    }

    function clearCutInfo() {
        hide(cutInfoDialog);
        cutInfoDialog.innerHTML = "";
    }

    /** 
     * @param {string} player
     * @param {Card[]} cards 
     * @param {number[]} positions
     * @param {number} timeout
     * @param {string} mask
     */
    function showCards(player, cards, positions, timeout = PEEK_TIMEOUT, mask = null) {
        const playerHand = PLAYERS[player].hand;
        const playerData = PLAYERS[player].data;
        drawHand(playerHand, player, playerData.cards_in_hand, cards, positions, mask)
        if (timeout > 0) {
            queueActionInstantly(async () => {
                await new Promise(r => setTimeout(r, timeout));
                clearPlayerHand(player);
            });
        }
    }

    function clearPlayersHands() {
        for (const [player, _] of Object.entries(PLAYERS)) {
            clearPlayerHand(player);
        }
    }

    function clearPlayerHand(player) {
        const p = PLAYERS[player];
        drawHand(p.hand, player, p.data.cards_in_hand, [], [], null);
    }

    /** 
     * @param {Element} container
     * @param {string} player
     * @param {number} cardsInHand
     * @param {Card[]} cards
     * @param {number[]} positions 
     * @param {string} mask
     */
    function drawHand(container, player, cardsInHand, cards, positions, mask = null) {
        while (container.firstChild) { container.removeChild(container.lastChild) }
        let cardix = 0;
        for (let i = 0; i < cardsInHand; i++) {
            let text;
            if (positions && positions[cardix] == i) {
                let value = mask ?? cardValue(cards[cardix]);
                text = document.createTextNode("[" + value + "]");
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
    async function showDraw(player, source, card, effect) {
        const playerDraw = PLAYERS[player].draw;
        setDrawScreen(player == THIS_PLAYER, effect);
        switch (source) {
            case "pile":
                await drawCard(deckPile, playerDraw);
                break;
            case "discard":
                await drawCard(deckDiscard, playerDraw);
                break;
            default:
                console.error("unknown draw source: ", source)
                break;
        }
        playerDraw.innerHTML = "";
        let text = card.suit ? "[" + cardValue(card) + "]" : "[ ]";
        if (effect && effect != "none") {
            text += " (Effect: " + EFFECTS[effect] + ")"
        }
        const node = createCardTemplate();
        node.appendChild(document.createTextNode(text));
        playerDraw.appendChild(node);
    }

    /** 
     * @param {string} player
     * @param {number} cardPosition
     * @param {Card} card
     */
    async function showDiscard(player, cardPosition, card) {
        const playerHand = PLAYERS[player].hand;
        const playerDraw = PLAYERS[player].draw;
        const cardInHand = /** @type {HTMLElement} */ (playerHand.childNodes[cardPosition]);
        const tmpContainer = createCardTemplate();
        setDiscardScreen();
        if (cardPosition >= 0) {
            const drawnCard = /** @type {HTMLElement} */ (playerDraw.lastChild);
            cardInHand.innerHTML = "[" + cardValue(card) + "]";
            cardInHand.onclick = () => sendAction({
                "type": "draw",
                "data": { "source": "discard" },
            });
            cardInHand.replaceWith(tmpContainer);
            tmpContainer.appendChild(cardInHand);
            await moveNode(cardInHand, deckDiscard);
            clearDiscardDeckExcept(cardInHand);
            await moveNode(drawnCard, tmpContainer);
            tmpContainer.replaceWith(drawnCard);
            drawnCard.innerHTML = "[ ]";
            drawnCard.onclick = () => sendCurrentAction(player, cardPosition);
        } else {
            const drawnCard = /** @type {HTMLElement} */ (playerDraw.lastChild);
            drawnCard.innerHTML = "[" + cardValue(card) + "]"
            drawnCard.onclick = () => sendAction({
                "type": "draw",
                "data": { "source": "discard" },
            });
            await moveNode(drawnCard, deckDiscard)
            clearDiscardDeckExcept(drawnCard);
        }
    }

    /** @param {HTMLElement} card */
    function clearDiscardDeckExcept(card) {
        while (deckDiscard.firstChild && deckDiscard.firstChild !== card) {
            deckDiscard.removeChild(deckDiscard.firstChild);
        }
    }

    /** 
     * @param {string} player
     * @param {number} cardPosition
     * @param {Card} card
     */
    async function showExtraDiscard(player, cardPosition, card) {
        if (cardPosition < 0) {
            console.error("extra discard cannot be the drawn card")
            return;
        }
        const playerHand = PLAYERS[player].hand;
        const cardInHand = /** @type {HTMLElement} */ (playerHand.childNodes[cardPosition]);
        cardInHand.innerHTML = "[" + cardValue(card) + "]"
        cardInHand.onclick = () => sendAction({
            "type": "draw",
            "data": { "source": "discard" },
        });
        await moveNode(cardInHand, deckDiscard)
        clearDiscardDeckExcept(cardInHand);
        playerHand.childNodes.forEach((_, ix) => {
            if (ix >= cardPosition) {
                let card = /** @type {HTMLElement} */ (playerHand.childNodes[ix]);
                card.onclick = () => sendCurrentAction(player, ix);
            }
        });
    }

    /**
     * @param {string} player
     * @param {number[]} cardPositions
     * @param {Card[]} cards
     * @param {Card} topOfDiscard
     */
    async function showFailedDoubleDiscard(player, cardPositions, cards, topOfDiscard) {
        const playerHand = PLAYERS[player].hand;
        const playerDraw = PLAYERS[player].draw;
        const tmpContainer = createCardTemplate();
        const drawnCard = /** @type {HTMLElement} */ (playerDraw.lastChild);

        // everyone gets to see the card values
        for (let ix = 0; ix < cardPositions.length; ix++) {
            const cardInHand = /** @type {HTMLElement} */ (playerHand.childNodes[cardPositions[ix]]);
            cardInHand.innerHTML = "[" + cardValue(cards[ix]) + "]";
        }

        // move drawn card to hand last position
        playerHand.appendChild(tmpContainer);
        await moveNode(drawnCard, tmpContainer)
        tmpContainer.replaceWith(drawnCard);
        drawnCard.onclick = () => sendCurrentAction(player, playerHand.childNodes.length - 1);
        PLAYERS[player].data.cards_in_hand += 1;

        // wait and hide cards
        await new Promise(r => setTimeout(r, 1000));
        drawnCard.innerHTML = "[ ]";
        for (let ix = 0; ix < cardPositions.length; ix++) {
            const cardInHand = /** @type {HTMLElement} */ (playerHand.childNodes[cardPositions[ix]]);
            cardInHand.innerHTML = "[ ]";
        }

        // set new discarded card
        // TODO: Should be done with `await drawCard(deckPile, playerDraw)` animation from draw pile if discard pile was not empty before draw.
        setLastDiscarded(topOfDiscard);
    }

    /** 
     * @param {string[]} players
     * @param {number[]} cardPositions 
     */
    async function showSwap(players, cardPositions) {
        const playerOneHand = PLAYERS[players[0]].hand;
        const playerOneCard = /** @type {HTMLElement} */ (playerOneHand.childNodes[cardPositions[0]]);
        const playerTwoHand = PLAYERS[players[1]].hand;
        const playerTwoCard = /** @type {HTMLElement} */ (playerTwoHand.childNodes[cardPositions[1]]);
        const tmpContainerOne = createCardTemplate();
        const tmpContainerTwo = createCardTemplate();


        playerOneCard.replaceWith(tmpContainerOne);
        tmpContainerOne.appendChild(playerOneCard);
        playerTwoCard.replaceWith(tmpContainerTwo)
        tmpContainerTwo.appendChild(playerTwoCard);

        await moveNode(playerOneCard, tmpContainerTwo, SWAP_DURATION);
        await moveNode(playerTwoCard, tmpContainerOne, SWAP_DURATION);

        tmpContainerOne.replaceWith(playerTwoCard);
        tmpContainerTwo.replaceWith(playerOneCard);
        playerOneCard.onclick = () => sendCurrentAction(players[0], cardPositions[0]);
        playerTwoCard.onclick = () => sendCurrentAction(players[1], cardPositions[1]);
    }

    /** 
     * @param {Player[]} players
     * @param {string} player
     * @param {boolean} withCount
     * @param {number} declared 
     * @param {Card[][]} hands
     */
    async function showCut(players, player, withCount, declared, hands) {
        setCutScreen();
        setPlayers(players);
        for (const [ix, [player, data]] of Object.entries(PLAYERS).entries()) {
            const positions = [...Array(hands[ix].length).keys()];
            showCards(player, hands[ix], positions, 0);
        }
        setCutInfo(player, withCount, declared);
        await waitUserInput();
        clearPlayersHands();
        clearCutInfo();
        setPlayers(players);
    }

    /** @param {Round[]} rounds */
    function showEndGame(rounds) {
        console.log(rounds);
        hide(gameContainer);

        const tbl = document.createElement("table");
        const tblHead = document.createElement("thead");
        const headRow = document.createElement("tr");
        const tblBody = document.createElement("tbody");

        const players = Object.keys(rounds[0].scores);

        // create headers
        const headCell = document.createElement("th");
        headCell.appendChild(document.createTextNode("Round"));
        headRow.appendChild(headCell);
        for (const playerID of players) {
            const cell = document.createElement("th");
            cell.appendChild(document.createTextNode(playerID));
            headRow.appendChild(cell);
        }

        // populate table
        for (let rIx = 0; rIx < rounds.length; rIx++) {
            const round = rounds[rIx];
            const row = document.createElement("tr");

            const roundCell = document.createElement("td");
            roundCell.appendChild(document.createTextNode("Round " + (rIx + 1)));
            row.appendChild(roundCell);

            for (const playerID of players) {
                const cell = document.createElement("td");
                cell.className = "score-cell";

                const text = document.createTextNode("" + round.scores[playerID]);
                if (playerID == round.cutter) {
                    cell.className += " cutter";
                    // TODO: Add cut withCount and declared info somewhere
                }

                const handContainer = document.createElement("div");
                handContainer.className = "hand-container"
                hide(handContainer);

                let containerText;
                if (round.hands[playerID].length > 0) {
                    containerText = document.createTextNode("Hand: " + round.hands[playerID].map(cardValue).join(" "));
                } else {
                    console.error("missing hand data for player: ", playerID, rounds)
                    containerText = document.createTextNode("missing hand data");
                }

                handContainer.appendChild(containerText);

                cell.onmouseover = () => show(handContainer, "flex");
                cell.onmouseout = () => hide(handContainer);

                cell.appendChild(text);
                cell.appendChild(handContainer);
                row.appendChild(cell);
            }


            tblBody.appendChild(row);
        }

        tblHead.appendChild(headRow);
        tbl.appendChild(tblHead);
        tbl.appendChild(tblBody);
        endgameContainer.appendChild(tbl);
    }


    /** @param {string} action */
    function setAction(action) {
        // TODO: check if action is valid
        console.log("Setting action to: ", action)
        CURRENT_ACTION = action
    }

    /** @param {UpdatePlayersChangedData} data */
    async function handlePlayersChanged(data) {
        setPlayers(data.players)
    }

    /** @param {UpdateStartNextRoundData} data */
    async function handleGameStart(data) {
        FIRST_TURN = true;
        setStartGameScreen();
        setPlayers(data.players);
    }

    /** @param {UpdatePlayerFirstPeekedData} data */
    async function handlePlayerPeeked(data) {
        if (data.player == THIS_PLAYER) {
            setPlayerPeekedScreen()
            showCards(data.player, data.cards, [0, 1], 0)
            await waitUserInput();
            clearPlayerHand(data.player);
        }
        markReady(data.player)
    }

    /** @param {UpdateTurnData} data */
    async function handleTurn(data) {
        if (FIRST_TURN) {
            clearCheckmarks();
            FIRST_TURN = false;
        }
        setTurnScreen(data.player == THIS_PLAYER);
    }

    /** @param {UpdateDrawData} data */
    async function handleDraw(data) {
        await showDraw(data.player, data.source, data.card, data.effect);
    }

    /** @param {UpdateDiscardData} data */
    async function handleDiscard(data) {
        for (let ix = 0; ix < data.cards.length; ix++) {
            if (ix == 0) {
                await showDiscard(data.player, data.cardsPositions[ix], data.cards[ix])
            } else {
                await showExtraDiscard(data.player, data.cardsPositions[1], data.cards[1]);
                PLAYERS[data.player].data.cards_in_hand -= 1;
            }
        }
    }

    /** @param {UpdateTypeFailedDoubleDiscardData} data */
    async function handleDoubleDiscard(data) {
        await showFailedDoubleDiscard(data.player, data.cardsPositions, data.cards, data.topOfDiscard);
    }

    /** @param {UpdatePeekCardData} data */
    async function handleEffectPeek(data) {
        if (data.card.value != 0 && data.card.suit != "") {
            showCards(data.player, [data.card], [data.cardPosition]);
        } else {
            showCards(data.player, [], [data.cardPosition], PEEK_TIMEOUT, "👁");
        }
    }

    /** @param {UpdateSwapCardsData} data */
    async function handleEffectSwap(data) {
        await showSwap(data.players, data.cardsPositions)
    }

    /** @param {UpdateCutData} data */
    async function handleCut(data) {
        await showCut(data.players, data.player, data.withCount, data.declared, data.hands);
    }

    /** @param {UpdateStartNextRoundData} data */
    async function handleNextRound(data) {
        FIRST_TURN = true;
        setStartRoundScreen();
        setPlayers(data.players)
    }

    /** @param {UpdateEndGameData} data */
    async function handleEndGame(data) {
        showEndGame(data.rounds);
    }

    /** @param {UpdateRejoinStateData} data */
    async function handleRejoinState(data) {
        setStartGameScreen();
        setPlayers(data.players);
        if (data.lastDiscarded) {
            setLastDiscarded(data.lastDiscarded)
        }
        if (data.cardInHand) {
            setCardInDraw(data.currentTurn, data.cardInHandValue)
        }
        if (data.currentTurn == THIS_PLAYER && data.cardInHand) {
            setDrawScreen(true, "none");
        } else {
            setTurnScreen(data.currentTurn == THIS_PLAYER);
        }
    }

    /** @param {MessageEvent<any>} event} */
    function processWSMessage(event) {
        const data = JSON.parse(event.data)
        const msgData = data.data;
        console.log("Received message:", data)
        switch (data.type) {
            case "players_changed":
                queueActions(async () => await handlePlayersChanged(msgData));
                break;
            case "game_start":
                queueActions(async () => await handleGameStart(msgData));
                break;
            case "player_peeked":
                queueActions(async () => await handlePlayerPeeked(msgData));
                break;
            case "turn":
                queueActions(async () => await handleTurn(msgData));
                break;
            case "draw":
                queueActions(async () => await handleDraw(msgData));
                break;
            case "discard":
                queueActions(async () => await handleDiscard(msgData));
                break;
            case "failed_double_discard":
                queueActions(async () => await handleDoubleDiscard(msgData));
                break;
            case "effect_peek":
                queueActions(async () => await handleEffectPeek(msgData));
                break;
            case "effect_swap":
                queueActions(async () => await handleEffectSwap(msgData));
                break;
            case "cut":
                queueActions(async () => await handleCut(msgData));
                break;
            case "start_next_round":
                queueActions(async () => await handleNextRound(msgData));
                break;
            case "end_game":
                queueActions(async () => await handleEndGame(msgData));
                break;
            case "rejoin_state":
                queueActions(async () => await handleRejoinState(msgData));
            default:
                console.error("Unknown message type", data.type, msgData)
                break;
        }
    }

    function rejoinLastRoomIfAny() {
        let sess_cookie = document.cookie.split(';').find(c => c.trim().startsWith('session_token='));
        if (!sess_cookie) {
            return;
        }
        let [playerID, roomID, _] = sess_cookie.split('=')[1].split('::');
        if (!playerID || !roomID) {
            return;
        }
        connectToRoom(playerID, roomID, "");
    }

    /** @param {string | null} message */
    function setError(message) {
        if (message) {
            errorContainer.innerHTML = message;
            show(errorContainer);
        } else {
            hide(errorContainer);
        }
    }

    function setTitle(message) {
        var style = /** @type {HTMLElement} */(document.querySelector('.main')).style;
        style.setProperty('--title', '"' + message + '"');
    }

    /** 
     * @param {string} username 
     * @param {string} roomid 
     * @param {string} password 
     * */
    function connectToRoom(username, roomid, password) {
        if (!roomid || !username) {
            return false;
        }
        conn = new WebSocket("ws://" + location.host + "/join?room=" + roomid + "&player=" + username + "&password=" + password);
        conn.onerror = () => setError("Error connecting to room");
        conn.onclose = () => {
            console.log("connection closed");
            show(menuContainer, "flex");
            hide(buttonStart);
            hide(buttonAddBot);
            hide(selectBotDiff);
            setTitle("Tincholi");
        }
        conn.onmessage = processWSMessage;
        conn.onopen = () => {
            setError(null);
            hide(menuContainer)
            show(buttonStart);
            show(buttonAddBot);
            show(selectBotDiff);
            setTitle("ROOM CODE: " + roomid)
            console.log("connected to room " + roomid);
            THIS_PLAYER = username;
            THIS_ROOM = roomid;
        }
        return false;
    }

    buttonShowCreateMenu.onclick = () => {
        hide(mainMenu);
        show(menuCreate, "flex");
    }

    buttonShowJoinMenu.onclick = () => {
        hide(mainMenu);
        show(menuJoin, "flex");
    }

    buttonNewRoom.onclick = async () => {
        const password = createMenuPassword.value;
        await fetch("http://" + location.host + "/new", {
            method: "POST",
            body: JSON.stringify({
                "max_players": parseInt(createMenuMaxPlayers.value),
                "password": password,
                "deck": {
                    "extended": createMenuUseExtendedDeck.checked,
                    "chaos": createMenuUseChaosDeck.checked,
                }
            }),
        })
            .then(response => response.text())
            .then(roomid => connectToRoom(createMenuUsername.value, roomid, password));
    };

    buttonJoinRooom.onclick = () => connectToRoom(joinMenuUsername.value, joinMenuRoomID.value, joinMenuPassword.value);

    buttonAddBot.onclick = () => {
        if (!THIS_ROOM) {
            return false;
        }
        const botdiff = selectBotDiff.options[selectBotDiff.selectedIndex].value;
        fetch("http://" + location.host + "/add-bot?difficulty=" + botdiff + "&room=" + THIS_ROOM)
            .then(response => response.text())
            .then(data => console.log(data));
        return false;
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
        "data": {
            "withCount": inputCutDeclare.checked,
            "declared": parseInt(inputCutDeclared.value)
        },
    });

    inputCutDeclare.onclick = () => {
        if (inputCutDeclare.checked) {
            show(inputCutDeclared);
        } else {
            hide(inputCutDeclared);
        }
    }

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

    startProcessingActions();
    rejoinLastRoomIfAny();
};