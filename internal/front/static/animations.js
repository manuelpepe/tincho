var ANIMATION_BUFFER = [];
var CURRENT_ANIMATION = null;

// TODO: Implement timeout param
// TODO: Accept an array of animations to make sure they are executed in order
export function queueAnimation(animation) {
    ANIMATION_BUFFER.push(animation);
}

export function startProcessingAnimations() {
    setInterval(() => {
        console.log("Processing animations...",)
        if (CURRENT_ANIMATION !== null) {
            console.log("animation already in progress", CURRENT_ANIMATION)
            return;
        }
        if (ANIMATION_BUFFER.length > 0) {
            var animation = ANIMATION_BUFFER.shift();
            CURRENT_ANIMATION = animation;
            console.log("running animation", animation)
            animation()
            console.log("animation finished", animation)
            CURRENT_ANIMATION = null;
        }
    }, 1000);
}

