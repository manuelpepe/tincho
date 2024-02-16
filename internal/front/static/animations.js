/**
 * @async
 * @callback animationCallback
 */

/** @type {animationCallback[]} */
var ANIMATION_BUFFER = [];

/** @type {animationCallback} */
var CURRENT_ANIMATION = null;



// TODO: Implement timeout param
/** @param {...animationCallback} animations */
export function queueAnimation(...animations) {
    ANIMATION_BUFFER.push(...animations);
}

export function startProcessingAnimations() {
    // requestAnimationFrame instead of setInterval?
    setInterval(async () => {
        if (CURRENT_ANIMATION !== null) {
            return;
        }
        if (ANIMATION_BUFFER.length > 0) {
            var animation = ANIMATION_BUFFER.shift();
            CURRENT_ANIMATION = animation;
            await animation();
            CURRENT_ANIMATION = null;
        }
    }, 1000);
}

export function queueAnimationInstantly(animation) {
    ANIMATION_BUFFER.unshift(animation);
}