/**
 * Gochi AR Link
 * Connects the AR module (ar.js) to the server's /api/ar endpoints, so the
 * pet placed in the room is the player's real pet: care given in AR goes
 * through the game's rules, tracked walks count as exercise, and the pet's
 * animation follows what it is doing.
 *
 * At a playdate (see /api/playdate), the first tap marks the playdate's
 * center: the spot all the players agreed on. Each pet has its own place
 * around the center; the other players' pets are shown where their games
 * say they are, and tapping one has your pet play with it.
 */

const GochiARLink = (function() {
    'use strict';

    // How often (ms) to report the camera and walk position, and to fetch
    // the pet's state
    const CAMERA_INTERVAL = 1000;
    const EXERCISE_INTERVAL = 1000;
    const POLL_INTERVAL = 2000;
    const PLAYDATE_INTERVAL = 1000;

    // Playdate places: pets stand around the center, this far from it (m)
    const PLAYDATE_RADIUS = 1.0;

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

    // Playdate state: the center marked in this AR view, and the last
    // state from the server
    let playdateTimer = null;
    let playdateCenter = null;
    let playdate = null;
    let playdateBusy = false;
    let inPlace = false; // The pet has been sent to its place

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
     * @param {function} opts.petLook - returns which of its species' sprites the pet shows (idle, happy, ...)
     * @param {function} opts.spriteFor - (species, look) returns the sprite another player's pet shows
     * @param {function} opts.petOutfit - returns the icons of what the pet wears
     * @param {function} opts.statusText - returns what the pet is doing
     * @param {function} opts.onEnd - called when AR closes
     * @param {boolean} opts.playdate - AR for the playdate the player is at
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
        GochiAR.on('onPetPlaced', pet => {
            placePet(pet.position);
            if (options.playdate) markCenter(pet.position);
        });
        GochiAR.on('onPetMoved', placePet);
        GochiAR.on('onPetInteraction', care);
        GochiAR.on('onCameraMove', reportCamera);
        GochiAR.on('onExerciseStart', () => call('/api/ar/exercise', { action: 'start' }));
        GochiAR.on('onExercisePosition', reportWalkPosition);
        GochiAR.on('onExerciseUpdate', finishWalk);
        GochiAR.on('onSessionEnd', end);

        if (!(await GochiAR.startSession())) {
            await call('/api/ar/session', { action: 'stop' });
            if (options.playdate) call('/api/playdate', { action: 'leave' });
            return false;
        }
        pollTimer = setInterval(syncPet, POLL_INTERVAL);
        if (options.playdate) startPlaydate();
        return true;
    }

    // ========================================================================
    // Playdates
    // ========================================================================

    /**
     * Show the playdate's panel and start exchanging positions
     */
    function startPlaydate() {
        playdateCenter = null;
        playdate = null;
        inPlace = false;
        const panel = document.createElement('div');
        panel.className = 'ar-playdate-panel';
        panel.id = 'ar-playdate-panel';
        document.getElementById('ar-overlay').append(panel);
        renderPlaydatePanel();
        updatePlaydate();
        playdateTimer = setInterval(updatePlaydate, PLAYDATE_INTERVAL);
    }

    /**
     * The place around the center for a playdate slot
     * @param {number} slot
     * @returns {{x: number, y: number, z: number}} Offset from the center
     */
    function slotOffset(slot) {
        const angle = slot * Math.PI / 3; // Six places
        return { x: PLAYDATE_RADIUS * Math.sin(angle), y: 0, z: PLAYDATE_RADIUS * Math.cos(angle) };
    }

    /**
     * Mark where the pet was placed as the playdate's center, and send
     * the pet to its place around it
     * @param {object} position - {x, y, z}
     */
    function markCenter(position) {
        if (playdateCenter) return;
        playdateCenter = { ...position };
        goToPlace();
        renderPlaydatePanel();
    }

    /**
     * Send the pet to its place around the center, once the center is
     * marked and the pet's place is known
     */
    function goToPlace() {
        const me = playdate && playdate.members.find(m => m.you);
        if (inPlace || !playdateCenter || !me) return;
        inPlace = true;
        const offset = slotOffset(me.slot);
        GochiAR.movePetTo({
            x: playdateCenter.x + offset.x,
            y: playdateCenter.y,
            z: playdateCenter.z + offset.z
        });
    }

    /**
     * Send the pet's position relative to the center (and how it looks)
     * and show the other pets
     */
    async function updatePlaydate() {
        if (playdateBusy) return;
        playdateBusy = true;
        const pet = GochiAR.getPlacedPet();
        let position = { x: 0, y: 0, z: 0 };
        if (pet && playdateCenter) {
            position = {
                x: pet.position.x - playdateCenter.x,
                y: pet.position.y - playdateCenter.y,
                z: pet.position.z - playdateCenter.z
            };
        }
        const result = await call('/api/playdate/update', {
            position,
            look: options.petLook ? options.petLook() : 'idle'
        });
        playdateBusy = false;
        if (!playdateTimer) return; // AR closed meanwhile
        if (!result.success) {
            message(result.error || 'The playdate ended', 'warning');
            clearInterval(playdateTimer);
            playdateTimer = null;
            playdate = null;
            renderPlaymates();
            renderPlaydatePanel(result.error || 'The playdate ended.');
            return;
        }
        playdate = result.data;
        goToPlace();
        renderPlaymates();
        renderPlaydatePanel();
    }

    /**
     * Show the playdate's code, who is there and what happened
     * @param {string} [ended] - Why the playdate ended
     */
    function renderPlaydatePanel(ended) {
        const panel = document.getElementById('ar-playdate-panel');
        if (!panel) return;
        panel.replaceChildren();
        const line = (text, className) => {
            const div = document.createElement('div');
            if (className) div.className = className;
            div.textContent = text;
            panel.append(div);
            return div;
        };
        if (ended) {
            line(ended);
            return;
        }
        if (!playdate) {
            line('Joining the playdate…');
            return;
        }
        const title = line('Playdate ');
        const code = document.createElement('span');
        code.className = 'ar-playdate-code';
        code.textContent = playdate.code;
        title.append(code, ' · ' + playdate.members.map(m => m.name).join(', '));
        if (!playdateCenter) {
            line('Stand where you can all see the same spot, and tap that spot to mark the playdate\'s center.',
                'ar-playdate-hint');
        } else if (playdate.members.length < 2) {
            line('Waiting for others. They join with the code ' + playdate.code + '.', 'ar-playdate-hint');
        } else {
            line('Tap another pet to have yours play with it.', 'ar-playdate-hint');
        }
        playdate.log.slice(-3).forEach(text => line(text, 'ar-playdate-log'));
    }

    /**
     * Show the other players' pets where they are around the center
     */
    function renderPlaymates() {
        const container = document.getElementById('ar-pet-container');
        if (!container) return;
        const others = playdate && playdateCenter ? playdate.members.filter(m => !m.you) : [];
        const shown = new Set();
        others.forEach(mate => {
            const id = 'ar-playmate-' + mate.slot;
            shown.add(id);
            let el = document.getElementById(id);
            if (!el) {
                el = document.createElement('div');
                el.className = 'ar-playmate';
                el.id = id;
                const sprite = document.createElement('div');
                sprite.className = 'pet-sprite';
                const name = document.createElement('div');
                name.className = 'ar-playmate-name';
                el.append(sprite, name);
                el.dataset.handle = mate.handle;
                el.addEventListener('click', () => playWith(el.dataset.handle));
                container.append(el);
            }
            el.dataset.handle = mate.handle;
            el.querySelector('.pet-sprite').textContent = options.spriteFor ? options.spriteFor(mate.species, mate.look) : '🐾';
            // What it wears, from the game's list of cosmetics
            const outfit = Object.values(mate.outfit || {}).map(c => c.icon).join('');
            el.querySelector('.ar-playmate-name').textContent = mate.name + (outfit ? ' ' + outfit : '') +
                (mate.relation ? ' · ' + mate.relation : '');
            el.title = mate.name;
            // Drawn like the AR module draws your pet
            const x = playdateCenter.x + mate.position.x;
            const z = playdateCenter.z + mate.position.z;
            el.style.transform = `translate(${x * 100}px, ${z * -50}px) scale(${GochiAR.config.petScale})`;
        });
        document.querySelectorAll('.ar-playmate').forEach(el => {
            if (!shown.has(el.id)) el.remove();
        });
    }

    /**
     * Have the player's pet play with another player's pet
     * @param {string} handle - The other player, as the hub calls them
     */
    async function playWith(handle) {
        const result = await call('/api/playdate/play', { handle });
        if (!result.success) {
            message(result.error || 'Something went wrong', 'warning');
            return;
        }
        GochiAR.setPetAnimation(result.data.quality >= 0.6 ? 'playing' : 'idle');
        if (options.onCare) {
            options.onCare(result.data);
        }
        updatePlaydate();
    }

    /**
     * Leave the playdate and remove its pets and panel
     */
    function endPlaydate() {
        clearInterval(playdateTimer);
        playdateTimer = null;
        playdate = null;
        playdateCenter = null;
        document.querySelectorAll('.ar-playmate, .ar-playdate-panel').forEach(el => el.remove());
        call('/api/playdate', { action: 'leave' });
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
        // What the pet wears, shown under it
        if (sprite && options.petOutfit) {
            let outfit = document.querySelector('#ar-pet .ar-outfit');
            if (!outfit) {
                outfit = document.createElement('div');
                outfit.className = 'ar-outfit';
                outfit.style.cssText = 'text-align:center;font-size:18px;line-height:1;';
                sprite.after(outfit);
            }
            outfit.textContent = options.petOutfit();
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
        if (options.playdate) endPlaydate();
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
