import { EFFECT_PEEK_CARTA_AJENA, EFFECT_PEEK_OWN, EFFECT_SWAP } from './constants.js';
import { createCardTemplate, hide, show } from './utils.js';


// TODO: Do away with repeated declarations 
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
const buttonSpeedToggle = document.getElementById("speed-toggle");

const cutUI = document.getElementById("cut-ui");
const inputCutDeclare = /** @type {HTMLInputElement} */ (document.getElementById("input-cut-declare"));
const inputCutDeclared = /** @type {HTMLInputElement} */ (document.getElementById("input-cut-declared"));

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
    hide(cutUI);
    hide(buttonDiscard);
    hide(buttonDiscardTwo);
    hide(buttonCancelDiscardTwo);
    hide(buttonSwap);
    hide(buttonPeekOwn);
    hide(buttonPeekCartaAjena);
}

export function setStartGameScreen() {
    hide(buttonStart);
    hide(buttonAddBot);
    hide(selectBotDiff);
    show(buttonSpeedToggle);
    show(buttonFirstPeek);
    show(deckPile, "flex");
    show(deckDiscard, "flex");
}

export function setPlayerPeekedScreen() {
    hideAllButtons();
}

/** @param {boolean} isCurPlayer */
export function setTurnScreen(isCurPlayer) {
    hideAllButtons();
    if (isCurPlayer) {
        show(buttonDraw);
        show(cutUI);
        inputCutDeclare.checked = false;
        inputCutDeclared.value = "0";
    }
}

/** 
 * @param {boolean} isCurPlayer
 * @param {string} effect 
 * @param {string} source */
export function setDrawScreen(isCurPlayer, effect, source) {
    hideAllButtons();
    if (isCurPlayer) {
        hide(buttonDraw);
        if (source == "pile") {
            show(buttonDiscard);
        }
        show(buttonDiscardTwo);
        hide(cutUI);
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
    let card = createCardTemplate();
    card.innerHTML = "[ ]"
    deckPile.append(card)
    deckDiscard.innerHTML = "";
    show(buttonFirstPeek)
}