var ANIMATION_BUFFER = [];
var CURRENT_ANIMATION = null;

// TODO: Implement timeout param
// TODO: Accept an array of animations to make sure they are executed in order
export function queueAnimation(animation) {
    ANIMATION_BUFFER.push(animation);
}

export function startProcessingAnimations() {
    // requestAnimationFrame instead of setInterval?
    setInterval(() => {
        if (CURRENT_ANIMATION !== null) {
            return;
        }
        if (ANIMATION_BUFFER.length > 0) {
            var animation = ANIMATION_BUFFER.shift();
            CURRENT_ANIMATION = animation;
            animation()
            CURRENT_ANIMATION = null;
        }
    }, 1000);
}

