/**
 * Gochi AR Link
 * Connects the AR module (ar.js) to the server's /api/ar endpoints, so the
 * pet placed in the room is the player's real pet: care given in AR goes
 * through the game's rules, tracked walks count as exercise, and the pet's
 * animation follows what it is doing.
 */

const GochiARLink = (function() {
    'use strict';

    // How often (ms) to report the camera and walk position, and to fetch
    // the pet's state
    const CAMERA_INTERVAL = 1000;
    const EXERCISE_INTERVAL = 1000;
    const POLL_INTERVAL = 2000;

    // AR care to the gesture the server turns into the same care
    const gestures = { pet: 'tap', play: 'double_tap', feed: 'long_press' };

    // Server animations to the AR module's animation classes
    const animations = {
        idle: 'idle', happy: 'happy', excited: 'excited', sleep: 'sleeping',
        eat: 'eating', play: 'playing', walk: 'walking', run: 'walking'
    };

    let options = {};
    let pollTimer = null;
    let lastCameraReport = 0;
    let lastExerciseReport = 0;

    /**
     * Send a request to the server
     * @param {string} path
     * @param {object} [body] - POSTed as JSON; without it the request is a GET
     * @returns {Promise<object>} The response ({success, data, error})
     */
    async function call(path, body) {
        const init = body === undefined ? {} : {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(body)
        };
        try {
            const response = await fetch(path, init);
            return await response.json();
        } catch (e) {
            console.error('[AR link]', path, e);
            return { success: false, error: 'Could not reach the game' };
        }
    }

    function message(text, kind = 'success') {
        if (options.onMessage) {
            options.onMessage(text, kind);
        }
    }

    /**
     * Start AR with the active pet
     * @param {object} opts
     * @param {function} opts.onMessage - (text, kind) shows a message
     * @param {function} opts.onCare - (result) shows the result of care
     * @param {function} opts.petSprite - returns the pet's current sprite
     * @param {function} opts.statusText - returns what the pet is doing
     * @param {function} opts.onEnd - called when AR closes
     * @returns {Promise<boolean>} Whether AR started
     */
    async function start(opts = {}) {
        options = opts;
        GochiAR.config.petSprite = opts.petSprite || null;

        // A session left over from a closed page is ended first
        await call('/api/ar/session', { action: 'stop' });
        const started = await call('/api/ar/session', { action: 'start' });
        if (!started.success) {
            message(started.error || 'Could not start AR', 'error');
            return false;
        }

        GochiAR.on('onMessage', message);
        GochiAR.on('onPlaneDetected', reportPlane);
        GochiAR.on('onPetPlaced', pet => placePet(pet.position));
        GochiAR.on('onPetMoved', placePet);
        GochiAR.on('onPetInteraction', care);
        GochiAR.on('onCameraMove', reportCamera);
        GochiAR.on('onExerciseStart', () => call('/api/ar/exercise', { action: 'start' }));
        GochiAR.on('onExercisePosition', reportWalkPosition);
        GochiAR.on('onExerciseUpdate', finishWalk);
        GochiAR.on('onSessionEnd', end);

        if (!(await GochiAR.startSession())) {
            await call('/api/ar/session', { action: 'stop' });
            return false;
        }
        pollTimer = setInterval(syncPet, POLL_INTERVAL);
        return true;
    }

    /**
     * Report a surface the AR module found
     * @param {object} plane - {id, type, center, extent}
     */
    async function reportPlane(plane) {
        const result = await call('/api/ar/plane', { plane });
        if (!result.success) {
            console.warn('[AR link] Surface not recorded:', result.error);
        }
    }

    /**
     * Place the pet (or move it) to a position in the room
     * @param {object} position - {x, y, z}
     */
    async function placePet(position) {
        const result = await call('/api/ar/place', { position });
        if (!result.success) {
            message(result.error || 'Could not place your pet', 'warning');
        }
    }

    /**
     * Give care in AR through the matching gesture
     * @param {string} type - pet, play or feed
     * @param {object} pet - The placed pet
     * @returns {Promise<boolean>} Whether the pet accepted the care
     */
    async function care(type, pet) {
        const gesture = gestures[type];
        if (!gesture) return true;

        const result = await call('/api/ar/gesture', { type: gesture, position: pet.position });
        if (!result.success) {
            message(result.error || 'Something went wrong', 'warning');
            return false;
        }
        if (!result.data.hit) {
            console.warn('[AR link] The gesture missed the pet');
            return false;
        }
        if (options.onCare) {
            options.onCare(result.data);
        }
        return true;
    }

    /**
     * Report the camera's pose, at most once per CAMERA_INTERVAL
     * @param {XRRigidTransform} transform
     */
    function reportCamera(transform) {
        const now = Date.now();
        if (now - lastCameraReport < CAMERA_INTERVAL) return;
        lastCameraReport = now;

        const p = transform.position;
        const o = transform.orientation;
        call('/api/ar/camera', {
            transform: {
                position: { x: p.x, y: p.y, z: p.z },
                rotation: { x: o.x, y: o.y, z: o.z, w: o.w },
                scale: { x: 1, y: 1, z: 1 }
            }
        });
    }

    /**
     * Report where the player is during a walk, at most once per
     * EXERCISE_INTERVAL
     * @param {object} position - {x, y, z}
     */
    function reportWalkPosition(position) {
        const now = Date.now();
        if (now - lastExerciseReport < EXERCISE_INTERVAL) return;
        lastExerciseReport = now;
        call('/api/ar/exercise', { action: 'position', position });
    }

    /**
     * End a walk; the server counts it towards the pet's exercise
     */
    async function finishWalk() {
        const result = await call('/api/ar/exercise', { action: 'stop' });
        message(result.success ? result.data.text : (result.error || 'Something went wrong'),
            result.success ? 'success' : 'warning');
    }

    /**
     * Show the pet's current sprite, animation and activity
     */
    async function syncPet() {
        const result = await call('/api/ar');
        if (!result.success || !GochiAR.getPlacedPet()) return;

        const placed = Object.values(result.data.scene.placed_pets || {})[0];
        const pet = GochiAR.getPlacedPet();
        if (placed && !pet.isMoving && animations[placed.animation]) {
            GochiAR.setPetAnimation(animations[placed.animation]);
        }

        const sprite = document.querySelector('#ar-pet .pet-sprite');
        if (sprite && options.petSprite) {
            sprite.textContent = options.petSprite();
        }
        const status = document.querySelector('.ar-status-text');
        if (status && options.statusText) {
            status.textContent = options.statusText();
        }
    }

    /**
     * Clean up when the AR session ends
     */
    function end() {
        clearInterval(pollTimer);
        pollTimer = null;
        call('/api/ar/session', { action: 'stop' });
        if (options.onEnd) {
            options.onEnd();
        }
    }

    return { start };
})();

if (typeof module !== 'undefined' && module.exports) {
    module.exports = GochiARLink;
}
