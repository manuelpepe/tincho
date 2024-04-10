/** @type {number} */
export var CARDS_IN_DECK = 0;

/** @type {number} */
export var CARDS_IN_DRAW_PILE = 0;

/** @param {number} n */
export function setCardsInDeck(n) {
    CARDS_IN_DECK = n;
}

/** @param {number} n */
export function setCardsInDrawPile(n) {
    CARDS_IN_DRAW_PILE = n;
}

export function subtractFromDrawPileCount() {
    CARDS_IN_DRAW_PILE -= 1;
}

/** @param {number} playerCount */
export function resetDrawPileCount(playerCount) {
    // -1 for card already in discard
    CARDS_IN_DRAW_PILE = CARDS_IN_DECK - 1 - (4 * playerCount)
}