/**
 * Gochi AR Module
 * Handles augmented reality features for placing pets in the real world
 */

const GochiAR = (function() {
    'use strict';

    // ========================================================================
    // Configuration
    // ========================================================================

    const config = {
        // Session settings
        trackingMode: 'world',  // world, orientation, image, none
        planeDetection: true,
        lightEstimation: true,

        // Pet settings
        petScale: 0.3,  // Base scale for pet in AR
        petHeightOffset: 0.05,  // Height above surface
        movementSpeed: 0.5,  // meters per second

        // Interaction settings
        tapThreshold: 300,  // ms for tap vs long press
        swipeThreshold: 50,  // pixels for swipe detection

        // Exercise tracking
        stepLength: 0.7,  // meters per step estimate
        caloriesPerStep: 0.04
    };

    // ========================================================================
    // State
    // ========================================================================

    let isSupported = false;
    let isActive = false;
    let xrSession = null;
    let xrRefSpace = null;
    let gl = null;
    let renderer = null;

    // AR state
    let sessionState = 'inactive';
    let trackingQuality = 0;
    let detectedPlanes = new Map();
    let placedPet = null;
    let cameraPosition = { x: 0, y: 0, z: 0 };

    // Touch handling
    let touchStartTime = 0;
    let touchStartPos = { x: 0, y: 0 };
    let isTouching = false;

    // Exercise tracking
    let isExerciseTracking = false;
    let exerciseData = {
        startTime: null,
        distance: 0,
        steps: 0,
        calories: 0,
        positions: []
    };

    // Callbacks
    const callbacks = {
        onSessionStart: null,
        onSessionEnd: null,
        onTrackingChange: null,
        onPlaneDetected: null,
        onPetPlaced: null,
        onPetInteraction: null,
        onExerciseUpdate: null,
        onError: null
    };

    // DOM Elements
    let arContainer = null;
    let arCanvas = null;
    let arOverlay = null;

    // ========================================================================
    // Initialization
    // ========================================================================

    /**
     * Initialize AR module
     * @returns {Promise<boolean>} Whether AR is supported
     */
    async function init() {
        console.log('[AR] Initializing...');

        // Check WebXR support
        isSupported = await checkARSupport();

        if (!isSupported) {
            console.log('[AR] WebXR AR not supported, using fallback mode');
        }

        createARElements();
        setupEventListeners();

        return isSupported;
    }

    /**
     * Check AR support
     * @returns {Promise<boolean>}
     */
    async function checkARSupport() {
        if (!navigator.xr) {
            return false;
        }

        try {
            return await navigator.xr.isSessionSupported('immersive-ar');
        } catch (e) {
            console.warn('[AR] Error checking AR support:', e);
            return false;
        }
    }

    /**
     * Create AR DOM elements
     */
    function createARElements() {
        // Create AR container
        arContainer = document.createElement('div');
        arContainer.id = 'ar-container';
        arContainer.className = 'ar-container hidden';
        arContainer.innerHTML = `
            <canvas id="ar-canvas"></canvas>
            <div class="ar-overlay" id="ar-overlay">
                <div class="ar-header">
                    <button class="ar-btn ar-close-btn" id="ar-close">
                        <span>&times;</span>
                    </button>
                    <div class="ar-status" id="ar-status">
                        <span class="ar-status-indicator"></span>
                        <span class="ar-status-text">Initializing...</span>
                    </div>
                    <button class="ar-btn ar-settings-btn" id="ar-settings">
                        <span>&#9881;</span>
                    </button>
                </div>
                <div class="ar-pet-container" id="ar-pet-container">
                    <div class="ar-pet" id="ar-pet"></div>
                </div>
                <div class="ar-instructions" id="ar-instructions">
                    <p>Point your camera at a flat surface</p>
                    <p>Tap to place your pet</p>
                </div>
                <div class="ar-controls" id="ar-controls">
                    <button class="ar-action-btn" data-action="feed" title="Feed">
                        <span>&#127829;</span>
                    </button>
                    <button class="ar-action-btn" data-action="play" title="Play">
                        <span>&#127934;</span>
                    </button>
                    <button class="ar-action-btn" data-action="call" title="Come here">
                        <span>&#128075;</span>
                    </button>
                    <button class="ar-action-btn" data-action="exercise" title="Exercise mode">
                        <span>&#127939;</span>
                    </button>
                    <button class="ar-action-btn" data-action="photo" title="Take photo">
                        <span>&#128247;</span>
                    </button>
                </div>
                <div class="ar-exercise-panel hidden" id="ar-exercise-panel">
                    <div class="exercise-stats">
                        <div class="exercise-stat">
                            <span class="stat-value" id="exercise-distance">0</span>
                            <span class="stat-label">meters</span>
                        </div>
                        <div class="exercise-stat">
                            <span class="stat-value" id="exercise-steps">0</span>
                            <span class="stat-label">steps</span>
                        </div>
                        <div class="exercise-stat">
                            <span class="stat-value" id="exercise-calories">0</span>
                            <span class="stat-label">calories</span>
                        </div>
                        <div class="exercise-stat">
                            <span class="stat-value" id="exercise-time">0:00</span>
                            <span class="stat-label">time</span>
                        </div>
                    </div>
                    <button class="ar-btn" id="exercise-stop">Stop Exercise</button>
                </div>
                <div class="ar-plane-indicator hidden" id="ar-plane-indicator">
                    <div class="plane-reticle"></div>
                </div>
            </div>
        `;

        document.body.appendChild(arContainer);

        // Cache elements
        arCanvas = document.getElementById('ar-canvas');
        arOverlay = document.getElementById('ar-overlay');
    }

    /**
     * Setup event listeners
     */
    function setupEventListeners() {
        // Close button
        document.getElementById('ar-close')?.addEventListener('click', stopSession);

        // Action buttons
        document.querySelectorAll('.ar-action-btn').forEach(btn => {
            btn.addEventListener('click', (e) => {
                const action = btn.dataset.action;
                handleARAction(action);
            });
        });

        // Exercise stop button
        document.getElementById('exercise-stop')?.addEventListener('click', stopExerciseMode);

        // Touch events for interaction
        arOverlay?.addEventListener('touchstart', handleTouchStart, { passive: false });
        arOverlay?.addEventListener('touchmove', handleTouchMove, { passive: false });
        arOverlay?.addEventListener('touchend', handleTouchEnd);

        // Mouse events for desktop testing
        arOverlay?.addEventListener('mousedown', handleMouseDown);
        arOverlay?.addEventListener('mousemove', handleMouseMove);
        arOverlay?.addEventListener('mouseup', handleMouseUp);
    }

    // ========================================================================
    // Session Management
    // ========================================================================

    /**
     * Start AR session
     * @param {string} petId - Pet ID to place in AR
     * @returns {Promise<boolean>}
     */
    async function startSession(petId) {
        if (isActive) {
            console.warn('[AR] Session already active');
            return false;
        }

        console.log('[AR] Starting session...');

        try {
            if (isSupported) {
                await startWebXRSession();
            } else {
                startFallbackSession();
            }

            isActive = true;
            sessionState = 'initializing';
            updateStatus('Initializing...');

            // Show AR container
            arContainer.classList.remove('hidden');

            // Fire callback
            if (callbacks.onSessionStart) {
                callbacks.onSessionStart();
            }

            // Start with tracking state after delay
            setTimeout(() => {
                if (isActive) {
                    sessionState = 'tracking';
                    trackingQuality = 0.8;
                    updateStatus('Ready - Tap to place pet');
                    showInstructions();

                    if (callbacks.onTrackingChange) {
                        callbacks.onTrackingChange(sessionState, trackingQuality);
                    }
                }
            }, 1500);

            return true;

        } catch (error) {
            console.error('[AR] Failed to start session:', error);
            handleError(error);
            return false;
        }
    }

    /**
     * Start WebXR session
     */
    async function startWebXRSession() {
        const sessionInit = {
            requiredFeatures: ['local-floor'],
            optionalFeatures: ['dom-overlay', 'hit-test', 'plane-detection', 'light-estimation']
        };

        if (arOverlay) {
            sessionInit.domOverlay = { root: arOverlay };
        }

        xrSession = await navigator.xr.requestSession('immersive-ar', sessionInit);

        // Setup WebGL context
        gl = arCanvas.getContext('webgl', { xrCompatible: true });
        await gl.makeXRCompatible();

        xrSession.updateRenderState({
            baseLayer: new XRWebGLLayer(xrSession, gl)
        });

        // Get reference space
        xrRefSpace = await xrSession.requestReferenceSpace('local-floor');

        // Setup frame loop
        xrSession.requestAnimationFrame(onXRFrame);

        // Handle session end
        xrSession.addEventListener('end', () => {
            stopSession();
        });
    }

    /**
     * Start fallback session (2D overlay mode)
     */
    function startFallbackSession() {
        console.log('[AR] Starting fallback mode');

        // Show camera feed simulation or static background
        arContainer.classList.add('ar-fallback');

        // Simulate plane detection after delay
        setTimeout(() => {
            if (isActive) {
                // Add a simulated plane
                const plane = {
                    id: 'fallback-plane',
                    type: 'horizontal',
                    center: { x: 0, y: 0, z: -1 },
                    extent: { x: 2, y: 0, z: 2 }
                };
                detectedPlanes.set(plane.id, plane);
                showPlaneIndicator();

                if (callbacks.onPlaneDetected) {
                    callbacks.onPlaneDetected(plane);
                }
            }
        }, 2000);
    }

    /**
     * Stop AR session
     */
    async function stopSession() {
        if (!isActive) return;

        console.log('[AR] Stopping session...');

        isActive = false;
        sessionState = 'inactive';

        // Stop exercise if active
        if (isExerciseTracking) {
            stopExerciseMode();
        }

        // End XR session
        if (xrSession) {
            await xrSession.end();
            xrSession = null;
        }

        // Clear state
        detectedPlanes.clear();
        placedPet = null;
        trackingQuality = 0;

        // Hide AR container
        arContainer.classList.add('hidden');
        arContainer.classList.remove('ar-fallback');

        // Fire callback
        if (callbacks.onSessionEnd) {
            callbacks.onSessionEnd();
        }
    }

    /**
     * WebXR frame callback
     * @param {number} time
     * @param {XRFrame} frame
     */
    function onXRFrame(time, frame) {
        if (!isActive || !xrSession) return;

        xrSession.requestAnimationFrame(onXRFrame);

        const pose = frame.getViewerPose(xrRefSpace);
        if (!pose) return;

        // Update camera position
        const position = pose.transform.position;
        cameraPosition = { x: position.x, y: position.y, z: position.z };

        // Update exercise tracking
        if (isExerciseTracking) {
            updateExercisePosition(cameraPosition);
        }

        // Update pet position if moving
        if (placedPet && placedPet.isMoving) {
            updatePetMovement(time);
        }

        // Render frame
        renderFrame(frame);
    }

    /**
     * Render AR frame
     * @param {XRFrame} frame
     */
    function renderFrame(frame) {
        // In a full implementation, this would render 3D pet model
        // For now, we use DOM overlay for the pet
    }

    // ========================================================================
    // Pet Placement
    // ========================================================================

    /**
     * Place pet at screen coordinates
     * @param {number} x - Screen X
     * @param {number} y - Screen Y
     */
    function placePetAt(x, y) {
        if (sessionState !== 'tracking') {
            console.warn('[AR] Not tracking, cannot place pet');
            return;
        }

        // Find plane at position (simplified)
        let targetPlane = null;
        let targetPosition = { x: 0, y: 0, z: -1 };

        if (detectedPlanes.size > 0) {
            // Use first horizontal plane
            for (const plane of detectedPlanes.values()) {
                if (plane.type === 'horizontal') {
                    targetPlane = plane;
                    targetPosition = { ...plane.center };
                    targetPosition.y += config.petHeightOffset;
                    break;
                }
            }
        }

        if (!targetPlane && !arContainer.classList.contains('ar-fallback')) {
            showInstructions('No surface detected. Point at a flat surface.');
            return;
        }

        // Create or move pet
        if (!placedPet) {
            placedPet = {
                id: 'ar-pet',
                position: targetPosition,
                rotation: { x: 0, y: 0, z: 0, w: 1 },
                scale: config.petScale,
                animation: 'idle',
                isMoving: false,
                targetPosition: null
            };

            showPet();
            hideInstructions();

            if (callbacks.onPetPlaced) {
                callbacks.onPetPlaced(placedPet);
            }

            GochiUI.toast.success('Pet placed in AR!');

        } else {
            // Move pet to new position
            movePetTo(targetPosition);
        }

        // Hide plane indicator
        hidePlaneIndicator();
    }

    /**
     * Show pet in AR
     */
    function showPet() {
        const petContainer = document.getElementById('ar-pet-container');
        const petElement = document.getElementById('ar-pet');

        if (!petContainer || !petElement) return;

        petContainer.classList.add('visible');

        // Copy pet sprite from main view
        const mainPetSprite = document.getElementById('pet-sprite');
        if (mainPetSprite) {
            petElement.innerHTML = mainPetSprite.outerHTML;
        } else {
            // Fallback pet display
            petElement.innerHTML = `
                <div class="pet-sprite">
                    <div class="pet-body"></div>
                    <div class="pet-face">
                        <div class="pet-eyes">
                            <div class="eye left"></div>
                            <div class="eye right"></div>
                        </div>
                        <div class="pet-mouth"></div>
                    </div>
                </div>
            `;
        }

        // Apply AR scale
        petElement.style.transform = `scale(${placedPet.scale})`;
    }

    /**
     * Move pet to target position
     * @param {object} target - Target position {x, y, z}
     */
    function movePetTo(target) {
        if (!placedPet) return;

        placedPet.targetPosition = target;
        placedPet.isMoving = true;
        setPetAnimation('walking');

        // Animate movement
        animatePetMovement();
    }

    /**
     * Animate pet movement
     */
    function animatePetMovement() {
        if (!placedPet || !placedPet.isMoving || !placedPet.targetPosition) {
            return;
        }

        const petElement = document.getElementById('ar-pet');
        if (!petElement) return;

        // Calculate direction
        const dx = placedPet.targetPosition.x - placedPet.position.x;
        const dz = placedPet.targetPosition.z - placedPet.position.z;
        const distance = Math.sqrt(dx * dx + dz * dz);

        // Check if arrived
        if (distance < 0.05) {
            placedPet.position = { ...placedPet.targetPosition };
            placedPet.targetPosition = null;
            placedPet.isMoving = false;
            setPetAnimation('idle');
            return;
        }

        // Move step
        const step = config.movementSpeed * 0.016; // Assuming 60fps
        const ratio = Math.min(step / distance, 1);

        placedPet.position.x += dx * ratio;
        placedPet.position.z += dz * ratio;

        // Update visual position (simplified 2D projection)
        const screenX = placedPet.position.x * 100;
        const screenY = placedPet.position.z * -50;
        petElement.style.transform = `translate(${screenX}px, ${screenY}px) scale(${placedPet.scale})`;

        // Continue animation
        requestAnimationFrame(animatePetMovement);
    }

    /**
     * Update pet movement in XR frame
     * @param {number} time
     */
    function updatePetMovement(time) {
        // Movement is handled by animatePetMovement for DOM-based rendering
    }

    /**
     * Set pet animation
     * @param {string} animation
     */
    function setPetAnimation(animation) {
        if (!placedPet) return;

        placedPet.animation = animation;

        const petElement = document.getElementById('ar-pet');
        if (!petElement) return;

        const sprite = petElement.querySelector('.pet-sprite');
        if (sprite) {
            // Remove all animation classes
            sprite.classList.remove('idle', 'walking', 'happy', 'excited', 'eating', 'playing', 'sleeping');
            sprite.classList.add(animation);
        }
    }

    // ========================================================================
    // Interactions
    // ========================================================================

    /**
     * Handle AR action button
     * @param {string} action
     */
    function handleARAction(action) {
        if (!placedPet) {
            GochiUI.toast.warning('Place your pet first!');
            return;
        }

        switch (action) {
            case 'feed':
                triggerPetInteraction('feed');
                setPetAnimation('eating');
                showAREffect('feed');
                break;

            case 'play':
                triggerPetInteraction('play');
                setPetAnimation('playing');
                showAREffect('play');
                break;

            case 'call':
                // Pet comes to camera position
                const targetPos = {
                    x: cameraPosition.x,
                    y: placedPet.position.y,
                    z: cameraPosition.z + 0.5 // In front of camera
                };
                movePetTo(targetPos);
                setPetAnimation('excited');
                break;

            case 'exercise':
                toggleExerciseMode();
                break;

            case 'photo':
                takeARPhoto();
                break;
        }
    }

    /**
     * Trigger pet interaction
     * @param {string} type
     */
    function triggerPetInteraction(type) {
        if (callbacks.onPetInteraction) {
            callbacks.onPetInteraction(type, placedPet);
        }

        // Also trigger main app interaction
        document.dispatchEvent(new CustomEvent('interaction', {
            detail: { action: type, source: 'ar' }
        }));
    }

    /**
     * Show AR effect
     * @param {string} type
     */
    function showAREffect(type) {
        const effectEmojis = {
            feed: ['🍖', '🥕', '🍎'],
            play: ['⚽', '🎾', '🎮'],
            pet: ['💕', '💖', '✨'],
            happy: ['😊', '💖', '✨']
        };

        const emojis = effectEmojis[type] || effectEmojis.happy;
        const petElement = document.getElementById('ar-pet');

        if (!petElement) return;

        // Create floating effects
        for (let i = 0; i < 3; i++) {
            setTimeout(() => {
                const effect = document.createElement('div');
                effect.className = 'ar-effect';
                effect.textContent = emojis[i % emojis.length];
                effect.style.left = `${50 + (Math.random() - 0.5) * 40}%`;
                effect.style.top = `${30 + Math.random() * 20}%`;
                petElement.appendChild(effect);

                setTimeout(() => effect.remove(), 1000);
            }, i * 200);
        }

        // Reset animation after delay
        setTimeout(() => {
            if (placedPet && !placedPet.isMoving) {
                setPetAnimation('idle');
            }
        }, 3000);
    }

    // ========================================================================
    // Touch/Mouse Handling
    // ========================================================================

    /**
     * Handle touch start
     * @param {TouchEvent} e
     */
    function handleTouchStart(e) {
        if (e.target.closest('.ar-btn, .ar-action-btn, .ar-controls')) {
            return; // Ignore touches on buttons
        }

        e.preventDefault();
        const touch = e.touches[0];
        touchStartTime = Date.now();
        touchStartPos = { x: touch.clientX, y: touch.clientY };
        isTouching = true;
    }

    /**
     * Handle touch move
     * @param {TouchEvent} e
     */
    function handleTouchMove(e) {
        if (!isTouching) return;
        e.preventDefault();

        const touch = e.touches[0];
        const dx = touch.clientX - touchStartPos.x;
        const dy = touch.clientY - touchStartPos.y;

        // Check for swipe
        if (Math.abs(dx) > config.swipeThreshold || Math.abs(dy) > config.swipeThreshold) {
            // This is a pan/swipe, not a tap
            if (placedPet) {
                // Could implement pet dragging here
            }
        }
    }

    /**
     * Handle touch end
     * @param {TouchEvent} e
     */
    function handleTouchEnd(e) {
        if (!isTouching) return;

        const duration = Date.now() - touchStartTime;

        if (duration < config.tapThreshold) {
            // This was a tap
            handleTap(touchStartPos.x, touchStartPos.y);
        } else if (duration > 500) {
            // Long press
            handleLongPress(touchStartPos.x, touchStartPos.y);
        }

        isTouching = false;
    }

    /**
     * Handle mouse events (for desktop testing)
     */
    function handleMouseDown(e) {
        if (e.target.closest('.ar-btn, .ar-action-btn, .ar-controls')) return;
        touchStartTime = Date.now();
        touchStartPos = { x: e.clientX, y: e.clientY };
        isTouching = true;
    }

    function handleMouseMove(e) {
        // Could implement dragging
    }

    function handleMouseUp(e) {
        if (!isTouching) return;
        const duration = Date.now() - touchStartTime;
        if (duration < config.tapThreshold) {
            handleTap(e.clientX, e.clientY);
        }
        isTouching = false;
    }

    /**
     * Handle tap
     * @param {number} x
     * @param {number} y
     */
    function handleTap(x, y) {
        if (!placedPet) {
            // First tap places the pet
            placePetAt(x, y);
        } else {
            // Check if tap is on pet
            const petElement = document.getElementById('ar-pet');
            if (petElement) {
                const rect = petElement.getBoundingClientRect();
                if (x >= rect.left && x <= rect.right && y >= rect.top && y <= rect.bottom) {
                    // Tap on pet - show affection
                    triggerPetInteraction('pet');
                    setPetAnimation('happy');
                    showAREffect('pet');
                } else {
                    // Tap elsewhere - move pet
                    // In real AR, this would use hit testing
                    // For now, move relative to current position
                    const centerX = window.innerWidth / 2;
                    const centerY = window.innerHeight / 2;
                    const offsetX = (x - centerX) / 200;
                    const offsetZ = (y - centerY) / 200;

                    movePetTo({
                        x: placedPet.position.x + offsetX,
                        y: placedPet.position.y,
                        z: placedPet.position.z + offsetZ
                    });
                }
            }
        }
    }

    /**
     * Handle long press
     * @param {number} x
     * @param {number} y
     */
    function handleLongPress(x, y) {
        if (placedPet) {
            // Long press on pet could open menu
            showPetMenu(x, y);
        }
    }

    /**
     * Show pet context menu
     * @param {number} x
     * @param {number} y
     */
    function showPetMenu(x, y) {
        // Could implement a radial menu here
        console.log('[AR] Pet menu at', x, y);
    }

    // ========================================================================
    // Exercise Mode
    // ========================================================================

    /**
     * Toggle exercise mode
     */
    function toggleExerciseMode() {
        if (isExerciseTracking) {
            stopExerciseMode();
        } else {
            startExerciseMode();
        }
    }

    /**
     * Start exercise mode
     */
    function startExerciseMode() {
        isExerciseTracking = true;
        exerciseData = {
            startTime: Date.now(),
            distance: 0,
            steps: 0,
            calories: 0,
            positions: [],
            lastPosition: null
        };

        // Show exercise panel
        document.getElementById('ar-exercise-panel')?.classList.remove('hidden');
        document.querySelector('.ar-action-btn[data-action="exercise"]')?.classList.add('active');

        // Pet follows user in exercise mode
        if (placedPet) {
            setPetAnimation('walking');
        }

        // Start timer update
        updateExerciseDisplay();

        GochiUI.toast.success('Exercise mode started! Walk around with your pet.');
    }

    /**
     * Stop exercise mode
     */
    function stopExerciseMode() {
        if (!isExerciseTracking) return;

        isExerciseTracking = false;

        // Hide exercise panel
        document.getElementById('ar-exercise-panel')?.classList.add('hidden');
        document.querySelector('.ar-action-btn[data-action="exercise"]')?.classList.remove('active');

        // Reset pet animation
        if (placedPet && !placedPet.isMoving) {
            setPetAnimation('idle');
        }

        // Calculate final stats
        const duration = (Date.now() - exerciseData.startTime) / 1000;
        const result = {
            duration: duration,
            distance: exerciseData.distance,
            steps: exerciseData.steps,
            calories: exerciseData.calories
        };

        // Fire callback
        if (callbacks.onExerciseUpdate) {
            callbacks.onExerciseUpdate(result);
        }

        // Show summary
        GochiUI.showModal({
            title: 'Exercise Complete!',
            content: `
                <div class="exercise-summary">
                    <p><strong>Time:</strong> ${formatTime(duration)}</p>
                    <p><strong>Distance:</strong> ${exerciseData.distance.toFixed(1)} meters</p>
                    <p><strong>Steps:</strong> ${exerciseData.steps}</p>
                    <p><strong>Calories:</strong> ${exerciseData.calories.toFixed(1)}</p>
                </div>
                <p>Great workout! Your pet enjoyed the walk too!</p>
            `,
            buttons: [{ text: 'OK', type: 'primary' }]
        });
    }

    /**
     * Update exercise position
     * @param {object} position
     */
    function updateExercisePosition(position) {
        if (!isExerciseTracking) return;

        if (exerciseData.lastPosition) {
            const dx = position.x - exerciseData.lastPosition.x;
            const dz = position.z - exerciseData.lastPosition.z;
            const distance = Math.sqrt(dx * dx + dz * dz);

            // Filter noise
            if (distance > 0.1) {
                exerciseData.distance += distance;
                exerciseData.positions.push({ ...position });

                // Estimate steps
                if (distance >= config.stepLength) {
                    const steps = Math.floor(distance / config.stepLength);
                    exerciseData.steps += steps;
                }

                // Estimate calories
                exerciseData.calories = exerciseData.steps * config.caloriesPerStep;

                // Pet follows at a distance
                if (placedPet && exerciseData.distance % 1 < 0.1) {
                    const followPos = {
                        x: position.x - 0.5,
                        y: placedPet.position.y,
                        z: position.z + 0.3
                    };
                    movePetTo(followPos);
                }
            }
        }

        exerciseData.lastPosition = { ...position };
        updateExerciseDisplay();
    }

    /**
     * Update exercise display
     */
    function updateExerciseDisplay() {
        if (!isExerciseTracking) return;

        const elapsed = (Date.now() - exerciseData.startTime) / 1000;

        document.getElementById('exercise-distance').textContent =
            exerciseData.distance.toFixed(1);
        document.getElementById('exercise-steps').textContent =
            exerciseData.steps;
        document.getElementById('exercise-calories').textContent =
            exerciseData.calories.toFixed(1);
        document.getElementById('exercise-time').textContent =
            formatTime(elapsed);

        // Continue updating
        if (isExerciseTracking) {
            setTimeout(updateExerciseDisplay, 1000);
        }
    }

    /**
     * Format time as MM:SS
     * @param {number} seconds
     * @returns {string}
     */
    function formatTime(seconds) {
        const mins = Math.floor(seconds / 60);
        const secs = Math.floor(seconds % 60);
        return `${mins}:${secs.toString().padStart(2, '0')}`;
    }

    // ========================================================================
    // AR Photo
    // ========================================================================

    /**
     * Take AR photo
     */
    async function takeARPhoto() {
        if (!placedPet) {
            GochiUI.toast.warning('Place your pet first!');
            return;
        }

        try {
            // Add flash effect
            arOverlay.classList.add('flash');
            setTimeout(() => arOverlay.classList.remove('flash'), 200);

            // In real implementation, capture WebXR frame
            // For now, show a toast
            GochiUI.toast.success('Photo saved!');

            // Pet reacts to photo
            setPetAnimation('happy');
            showAREffect('happy');

        } catch (error) {
            console.error('[AR] Photo failed:', error);
            GochiUI.toast.error('Failed to take photo');
        }
    }

    // ========================================================================
    // UI Helpers
    // ========================================================================

    /**
     * Update status display
     * @param {string} text
     */
    function updateStatus(text) {
        const statusText = document.querySelector('.ar-status-text');
        const statusIndicator = document.querySelector('.ar-status-indicator');

        if (statusText) {
            statusText.textContent = text;
        }

        if (statusIndicator) {
            statusIndicator.className = 'ar-status-indicator';
            if (sessionState === 'tracking') {
                statusIndicator.classList.add('tracking');
            } else if (sessionState === 'limited') {
                statusIndicator.classList.add('limited');
            } else if (sessionState === 'error') {
                statusIndicator.classList.add('error');
            }
        }
    }

    /**
     * Show instructions
     * @param {string} text
     */
    function showInstructions(text = null) {
        const instructions = document.getElementById('ar-instructions');
        if (instructions) {
            if (text) {
                instructions.querySelector('p').textContent = text;
            }
            instructions.classList.add('visible');
        }
    }

    /**
     * Hide instructions
     */
    function hideInstructions() {
        const instructions = document.getElementById('ar-instructions');
        if (instructions) {
            instructions.classList.remove('visible');
        }
    }

    /**
     * Show plane indicator
     */
    function showPlaneIndicator() {
        const indicator = document.getElementById('ar-plane-indicator');
        if (indicator) {
            indicator.classList.remove('hidden');
        }
    }

    /**
     * Hide plane indicator
     */
    function hidePlaneIndicator() {
        const indicator = document.getElementById('ar-plane-indicator');
        if (indicator) {
            indicator.classList.add('hidden');
        }
    }

    /**
     * Handle error
     * @param {Error} error
     */
    function handleError(error) {
        sessionState = 'error';
        updateStatus('Error: ' + error.message);

        if (callbacks.onError) {
            callbacks.onError(error);
        }
    }

    // ========================================================================
    // Callback Registration
    // ========================================================================

    /**
     * Set callback
     * @param {string} event
     * @param {function} callback
     */
    function on(event, callback) {
        if (callbacks.hasOwnProperty(event)) {
            callbacks[event] = callback;
        }
    }

    /**
     * Remove callback
     * @param {string} event
     */
    function off(event) {
        if (callbacks.hasOwnProperty(event)) {
            callbacks[event] = null;
        }
    }

    // ========================================================================
    // Public API
    // ========================================================================

    return {
        // Initialization
        init,
        isSupported: () => isSupported,
        isActive: () => isActive,

        // Session management
        startSession,
        stopSession,
        getSessionState: () => sessionState,
        getTrackingQuality: () => trackingQuality,

        // Pet
        placePetAt,
        movePetTo,
        setPetAnimation,
        getPlacedPet: () => placedPet,

        // Exercise
        startExerciseMode,
        stopExerciseMode,
        isExerciseTracking: () => isExerciseTracking,
        getExerciseData: () => ({ ...exerciseData }),

        // Photo
        takeARPhoto,

        // Callbacks
        on,
        off,

        // Configuration
        config
    };
})();

// Export for module systems
if (typeof module !== 'undefined' && module.exports) {
    module.exports = GochiAR;
}
