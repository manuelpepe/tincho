/** @type {number} */
export var CARDS_IN_DECK = 0;

/** @type {number} */
export var CARDS_IN_DRAW_PILE = 0;

export function setCardsInDeck(cardsInDeck) {
    CARDS_IN_DECK = cardsInDeck;
}

export function subtractFromDrawPileCount() {
    CARDS_IN_DRAW_PILE -= 1;
}

export function resetDrawPileCount(playerCount) {
    // -1 for card already in discard
    CARDS_IN_DRAW_PILE = CARDS_IN_DECK - 1 - (4 * playerCount)
}