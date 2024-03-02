/** @typedef {{suit: string, value: number}} Card */
/** @typedef {{id: string, points: number, pending_first_peek: boolean, cards_in_hand: number}} Player */
/** @typedef {{player: string, cardPosition: number}} SwapBuffer */
/** @typedef {{playerID: string, score: number}} PlayerScore */

/** @typedef {{players: Player[]}} UpdatePlayersChangedData */
/** @typedef {{players: Player[]}} UpdateStartNextRoundData */
/** @typedef {{player: string, cards: Card[]}} UpdatePlayerFirstPeekedData */
/** @typedef {{player: string}} UpdateTurnData */
/** @typedef {{player: string, source: string, card: Card, effect: string}} UpdateDrawData */
/** @typedef {{cardPosition: number, card: Card, player: string}} UpdatePeekCardData */
/** @typedef {{cardsPositions: number[], players: string[]}} UpdateSwapCardsData */
/** @typedef {{player: string, cardsPositions: number[], cards: Card[]}} UpdateDiscardData */
/** @typedef {{player: string, cardsPositions: number[], cards: Card[]}} UpdateTypeFailedDoubleDiscardData */
/** @typedef {{withCount: boolean, declared: number, player: string, players: Player[], hands: Card[][]}} UpdateCutData */
/** @typedef {{message: string}} UpdateErrorData */
/** @typedef {{scores: PlayerScore[][]}} UpdateEndGameData */
/** @typedef {{players: Player[], currentTurn: string, cardInHand: boolean, cardInHandValue: Card | null, lastDiscarded: Card | null}} UpdateRejoinStateData */
