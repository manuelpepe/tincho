export const SUITS = {
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

export const EFFECT_SWAP = "swap_card"
export const EFFECT_PEEK_OWN = "peek_own"
export const EFFECT_PEEK_CARTA_AJENA = "peek_carta_ajena"
export const ACTION_DISCARD = "discard"
export const ACTION_DISCARD_TWO = "discard_two"

export const EFFECTS = {
    [EFFECT_SWAP]: "Swap 2 cards",
    [EFFECT_PEEK_OWN]: "Peek card from your hand",
    [EFFECT_PEEK_CARTA_AJENA]: "Peek card from other player"
}