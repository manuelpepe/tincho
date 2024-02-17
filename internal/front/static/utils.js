/** @param {HTMLElement} node */
export function hide(node) {
    node.style.display = "none";
}

/** @param {HTMLElement} node */
export function show(node) {
    node.style.display = "block";
}

/** 
 * @param {Element} node
 * @param {Element} target 
 * @param {number | null} duration
 */
export async function moveNode(node, target, duration = 1000) {
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
    return new Promise(r => animation.finished.then(r))
}