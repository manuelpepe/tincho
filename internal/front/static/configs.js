const BASE_ANIMATION_DURATION = 1000;
const BASE_SWAP_DURATION = 2000;

var DURATION_MULTIPLIER = 1;
const VALID_DURATION_MULTIPLIERS = [1, 2];

export var PEEK_TIMEOUT = 10000;
export var ANIMATION_DURATION = BASE_ANIMATION_DURATION;
export var SWAP_DURATION = BASE_SWAP_DURATION;

const buttonSpeedToggle = document.getElementById("speed-toggle");

buttonSpeedToggle.addEventListener("click", () => {
    const idx = VALID_DURATION_MULTIPLIERS.indexOf(DURATION_MULTIPLIER);
    DURATION_MULTIPLIER = VALID_DURATION_MULTIPLIERS[(idx + 1) % VALID_DURATION_MULTIPLIERS.length];
    ANIMATION_DURATION = BASE_ANIMATION_DURATION / DURATION_MULTIPLIER;
    SWAP_DURATION = BASE_SWAP_DURATION / DURATION_MULTIPLIER;
    buttonSpeedToggle.textContent = `x${DURATION_MULTIPLIER}`;
});
