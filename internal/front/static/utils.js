import { ANIMATION_DURATION } from "./configs.js";

/** @param {HTMLElement} node */
export function hide(node) {
    node.style.display = "none";
}

/** @param {HTMLElement} node */
export function show(node, display = "block") {
    node.style.display = display;
}

/** 
 * @param {Element} node
 * @param {Element} target 
 * @param {number | null} duration
 */
export async function moveNode(node, target, duration = ANIMATION_DURATION) {
    if (duration == null || duration == undefined) {
        duration = ANIMATION_DURATION;
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
    return new Promise(r => animation.finished.then(r))
}

/** @returns {HTMLElement} */
export function createCardTemplate() {
    const card = document.createElement("div");
    card.className = "card";
    return card;
}

/** @returns {{wait: () => Promise<void>, notify: () => void}} */
export function getWaiter() {
    const timeout = async ms => new Promise(res => setTimeout(res, ms));
    let next = false;

    async function wait() {
        while (next === false) await timeout(50);
        next = false;
    }

    function notify() {
        next = true;
    }

    return {
        wait,
        notify,
    }
}
