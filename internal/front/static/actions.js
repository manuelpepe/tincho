/**
 * @async
 * @callback actionCallback
 */

/** @type {actionCallback[]} */
var ACTIONS_BUFFER = [];

/** @type {actionCallback} */
var CURRENT_ACTION = null;



// TODO: Implement timeout param
/** @param {...actionCallback} actions */
export function queueActions(...actions) {
    ACTIONS_BUFFER.push(...actions);
}

export function startProcessingActions() {
    // requestAnimationFrame instead of setInterval?
    setInterval(async () => {
        if (CURRENT_ACTION !== null) {
            return;
        }
        if (ACTIONS_BUFFER.length > 0) {
            var action = ACTIONS_BUFFER.shift();
            console.log('Processing action:', action);
            CURRENT_ACTION = action;
            await action();
            CURRENT_ACTION = null;
        }
    }, 1000);
}

export function queueActionInstantly(action) {
    ACTIONS_BUFFER.unshift(action);
}