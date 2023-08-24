window.onload = function () {
    /** @typedef {{suit: string, value: string}} Card */
    /** @typedef {{id: string, points: number, pending_first_peek: boolean, cards_in_hand: number}} Player */

    const SUITS = {
        "spanish": {
            "clubs": "B",
            "hearths": "C",
            "diamonds": "O",
            "spades": "E", 
        },
        "standard": {
            "clubs": "♧",
            "hearths": "♥",
            "diamonds": "♢",
            "spades": "♤", 
        }
    }
    var suitKind = "spanish"

    /** @param {Card} card */
    function cardValue(card) {
        return card.value + SUITS[card.suit]
    }

    /** @type {WebSocket} */
    var conn;

    /** @type {Object<string, {hand: HTMLElement, draw: HTMLElement, data: Player}>} */
    var players = {};

    const msg = document.getElementById("msg");
    const roomid = document.getElementById("room-id");
    const username = document.getElementById("username");

    const roomTitle = document.getElementById("room-title");
    const formJoin = document.getElementById("room-join");
    const formNew = document.getElementById("room-new");
    const buttonStart = document.getElementById("btn-start");
    const buttonFirstPeek = document.getElementById("btn-first-peek");
    const buttonDraw = document.getElementById("btn-draw");
    const buttonDiscard = document.getElementById("btn-discard");
    const buttonCut = document.getElementById("btn-cut");

    const playerTemplate = document.getElementById("player-template")
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

    /** @param {HTMLElement} node
        @param {HTMLElement} target */
    function moveNode(node, target) {
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
            duration: 1000,
            easing: 'linear',
        });
        return animation
    }

    /** @param {HTMLElement} sourcePile
        @param {HTMLElement} target */
    function drawCard(sourcePile, target) {
        const card = sourcePile.getElementsByClassName("card")[0];
        const cardClone = card.cloneNode(true);
        sourcePile.append(cardClone);
        return moveNode(card, target);
    }

    /** @param {Player[]} new_players */
    function setPlayers(new_players) {
        console.log("players", new_players)
        players = {};
        playerList.innerHTML = "";
        for (let i = 0; i < new_players.length; i++) {
            addPlayer(new_players[i]);
        }
    }

    /** @param {Player} player */
    function addPlayer(player) {
        let clone = playerTemplate.content.cloneNode(true);
        let parts = clone.querySelectorAll(".player-datafield");
        parts[0].innerHTML = player.id;
        drawHand(parts[1], player.cards_in_hand ?? 0, [], [])
        players[player["id"]] = {
            "hand": parts[1],
            "draw": parts[2],
            "data": player,
        };
        playerList.appendChild(clone);
    }

    /** @param {string} player
        @param {Card[]} cards 
        @param {number[]} positions*/
    function showCards(player, cards, positions) {
        const playerHand = players[player]["hand"];
        const playerData = players[player]["data"];
        drawHand(playerHand, playerData.cards_in_hand, cards, positions)
        // TODO: Store timeout for skip button
        setTimeout(() => {
            drawHand(playerHand, playerData.cards_in_hand, [], [])
        }, 3000);
    }

    /** @param {HTMLElement} container
        @param {number} cardsInHand
        @param {Card[]} cards
        @param {number[]} positions */
    function drawHand(container, cardsInHand, cards, positions) {
        while (container.firstChild) container.removeChild(container.lastChild)
        let cardix = 0;
        for (let i = 0; i < cardsInHand; i++) {
            let text;
            if (positions && positions[cardix] == i) {
                // TODO: change suit to suit letter in spanish deck
                text = document.createTextNode("[" + cardValue(cards[cardix]) + "]");
                cardix = cardix + 1
            } else {
                text = document.createTextNode("[ ]");
            }
            const node = document.createElement("div");
            node.className = "card";
            node.onclick = () => sendDiscard(i)
            node.appendChild(text);
            container.appendChild(node);
        }
    }

    /** @param {string} player */
    // eslint-disable-next-line no-unused-vars
    function markReady(player) { /* TODO */ }

    /** @param {string} player
        @param {string} source
        @param {Card} card
        @param {string} effect */
    function showDraw(player, source, card, effect) {
        // TODO: Draw from discard pile
        const playerDraw = players[player]["draw"];
        const animation = drawCard(deckPile, playerDraw);
        animation.addEventListener("finish", () => {
            playerDraw.innerHTML = "";
            let text = card.suit ? "[" + cardValue(card) + "]" : "[ ]";
            if (effect != "none") {
                // TODO: Pretty print effect
                text += " (" + effect + ")"
            }
            const textNode = document.createTextNode(text);
            const node = document.createElement("div");
            node.className = "card";
            node.appendChild(textNode);
            playerDraw.appendChild(node);
        });
    }

    function showDiscard(player, cardPosition, card) {
        const playerHand = players[player]["hand"];
        const playerDraw = players[player]["draw"];
        const newCard = playerDraw.lastChild;
        if (cardPosition >= 0) {
            const container = document.createElement("div");
            container.className = "card"
            const oldCard = playerHand.replaceChild(container, playerHand.childNodes[cardPosition]);
            container.appendChild(oldCard);
            const animation = moveNode(newCard, container);
            oldCard.innerHTML = "[" + cardValue(card) + "]";
            animation.addEventListener("finish", () => {
                moveNode(oldCard, deckDiscard)
                playerHand.replaceChild(newCard, container);
                newCard.innerHTML = "[ ]";
                newCard.onclick = () => sendDiscard(cardPosition);
            });
        } else {
            moveNode(newCard, deckDiscard)
        }
    }

    /** @param {MessageEvent<any>} event} */
    function processWSMessage(event) {
        const data = JSON.parse(event.data)
        const msgData = data.data;
        console.log("Received message:", data)
        if (data.type == "players_changed") {
            setPlayers(msgData.players)
        } else if (data.type == "game_start") {
            hide(buttonStart)
            show(buttonFirstPeek)
            show(deckPile)
            show(deckDiscard)
            setPlayers(msgData.players)
        } else if (data.type == "player_peeked") {
            if (msgData.player == username.value) {
                hide(buttonFirstPeek)
                showCards(msgData.player, msgData.cards, [0, 1])
            }
            markReady(msgData.player) // TODO
        } else if (data.type == "turn") {
            let fn = msgData.player == username.value ? show : hide
            fn(buttonDraw)
            fn(buttonDiscard)
            fn(buttonCut)
        } else if (data.type == "draw") {
            showDraw(msgData.player, msgData.source, msgData.card, msgData.effect)
        } else if (data.type == "discard") {
            showDiscard(msgData.player, msgData.cardPosition, msgData.card)
        } else if (data.type == "effect_peek") {
            showPeek(msgData.player, msgData.cardPosition, msgData.card) // TODO
        } else if (data.type == "effect_swap") {
            showSwap(msgData.players, msgData.cardPositions) // TODO
        } else if (data.type == "cut") {
            showCut(msgData.player, msgData.withCount, msgData.declared) // TODO
            setPlayers(msgData.players)
        } else if (data.type == "end_game") {
            showEndGame(msgData.winner)
        } else {
            console.error("Unknown message type", data.type, msgData)
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

    function sendDiscard(ix) {
        sendAction({
            "type": "discard",
            "data": { "cardPosition": ix },
        });
    }

    function sendAction(data) {
        if (!conn) {
            console.error("WS not connected");
            return false;
        }
        conn.send(JSON.stringify(data));
    }
};