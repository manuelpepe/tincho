import { EFFECT_PEEK_CARTA_AJENA, EFFECT_PEEK_OWN, EFFECT_SWAP } from './constants.js';
import { hide, show } from './utils.js';


// TODO: Do away with repeated declarations 
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

const deckPile = document.getElementById("deck-pile");
const deckDiscard = document.getElementById("deck-discard");


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

function hideAllButtons() {
    hide(buttonFirstPeek);
    hide(buttonDraw);
    hide(buttonCut);
    hide(buttonDiscard);
    hide(buttonDiscardTwo);
    hide(buttonCancelDiscardTwo);
    hide(buttonSwap);
    hide(buttonPeekOwn);
    hide(buttonPeekCartaAjena);
}

export function setStartGameScreen() {
    hide(buttonStart);
    show(buttonFirstPeek);
    show(deckPile);
    show(deckDiscard);
}

export function setPlayerPeekedScreen() {
    hideAllButtons();
}

/** @param {boolean} isCurPlayer */
export function setTurnScreen(isCurPlayer) {
    hideAllButtons();
    if (isCurPlayer) {
        show(buttonDraw);
        show(buttonCut);
    }
}

/** 
 * @param {boolean} isCurPlayer
 * @param {string} effect */
export function setDrawScreen(isCurPlayer, effect) {
    hideAllButtons();
    if (isCurPlayer) {
        hide(buttonDraw);
        show(buttonDiscard);
        show(buttonDiscardTwo);
        hide(buttonCut);
        showEffectButton(effect);
    }
}

export function setDiscardScreen() {
    hideAllButtons();
}

export function setCutScreen() {
    hideAllButtons();
}

export function setStartRoundScreen() {
    deckPile.innerHTML = "";
    deckDiscard.innerHTML = "";
    show(buttonFirstPeek)
}