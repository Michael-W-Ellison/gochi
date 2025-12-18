/**
 * Gochi Main Application
 * Initializes and coordinates all application modules
 */

const GochiApp = (function() {
    'use strict';

    // ========================================================================
    // Configuration
    // ========================================================================

    const config = {
        updateInterval: 60000,        // Update pet every minute
        statusPollInterval: 30000,    // Poll status every 30 seconds
        autoSaveInterval: 300000,     // Auto-save every 5 minutes
        reconnectOnFocus: true        // Reconnect WebSocket when tab gains focus
    };

    // ========================================================================
    // State
    // ========================================================================

    let currentPet = null;
    let currentUser = null;
    let updateTimer = null;
    let statusTimer = null;
    let autoSaveTimer = null;
    let isInitialized = false;
    let lastUpdateTime = Date.now();

    // ========================================================================
    // Initialization
    // ========================================================================

    /**
     * Initialize the application
     */
    async function init() {
        console.log('[Gochi] Initializing application...');

        try {
            // Initialize UI first
            GochiUI.init();
            GochiUI.showLoading('Starting Gochi...');

            // Initialize pet module
            GochiPet.init();

            // Check authentication
            if (GochiAPI.auth.isAuthenticated()) {
                await loadUserData();
            } else {
                // For demo purposes, skip auth and show onboarding
                GochiUI.hideLoading();
                await checkExistingPet();
            }

            // Setup event listeners
            setupEventListeners();

            // Setup visibility change handler
            setupVisibilityHandler();

            isInitialized = true;
            console.log('[Gochi] Application initialized successfully');

        } catch (error) {
            console.error('[Gochi] Initialization failed:', error);
            GochiUI.hideLoading();
            GochiUI.toast.error('Failed to initialize application');
        }
    }

    /**
     * Load user data after authentication
     */
    async function loadUserData() {
        try {
            currentUser = await GochiAPI.auth.getProfile();
            await loadPets();
        } catch (error) {
            console.error('[Gochi] Failed to load user data:', error);
            // Clear invalid tokens
            localStorage.removeItem('gochi_auth_token');
            localStorage.removeItem('gochi_refresh_token');
            GochiUI.showOnboarding();
        }
    }

    /**
     * Check for existing pet in local storage (demo mode)
     */
    async function checkExistingPet() {
        const savedPetId = localStorage.getItem('gochi_current_pet_id');

        if (savedPetId) {
            try {
                await loadPet(savedPetId);
            } catch (error) {
                console.log('[Gochi] No saved pet found, showing onboarding');
                GochiUI.showOnboarding();
            }
        } else {
            GochiUI.showOnboarding();
        }
    }

    /**
     * Load user's pets
     */
    async function loadPets() {
        try {
            const pets = await GochiAPI.pets.list();

            if (pets && pets.length > 0) {
                // Load the first pet (or previously selected)
                const savedPetId = localStorage.getItem('gochi_current_pet_id');
                const petToLoad = pets.find(p => p.id === savedPetId) || pets[0];
                await selectPet(petToLoad);
            } else {
                GochiUI.showOnboarding();
            }
        } catch (error) {
            console.error('[Gochi] Failed to load pets:', error);
            GochiUI.showOnboarding();
        }
    }

    /**
     * Load a specific pet
     * @param {string} petId - Pet ID to load
     */
    async function loadPet(petId) {
        GochiUI.showLoading('Loading pet...');

        try {
            const pet = await GochiAPI.pets.get(petId);
            await selectPet(pet);
        } catch (error) {
            console.error('[Gochi] Failed to load pet:', error);
            throw error;
        } finally {
            GochiUI.hideLoading();
        }
    }

    /**
     * Select and display a pet
     * @param {object} pet - Pet data
     */
    async function selectPet(pet) {
        currentPet = pet;

        // Save current pet ID
        localStorage.setItem('gochi_current_pet_id', pet.id);

        // Update pet display
        try {
            GochiPet.setPet(pet);
            GochiPet.updatePetInfo(pet);

            if (pet.vitals) {
                GochiPet.updateVitals(pet.vitals);
            }

            if (pet.emotions) {
                GochiPet.updateEmotions(pet.emotions);
            }

            GochiPet.updateStats(pet);
        } catch (e) {
            console.warn('[Gochi] Error updating pet display:', e);
        }

        // Initialize social module with pet ID (optional)
        try {
            GochiSocial.init(pet.id);
        } catch (e) {
            console.warn('[Gochi] Social module init failed:', e);
        }

        // Connect WebSocket (optional - app works without it)
        try {
            await connectWebSocket();
        } catch (e) {
            console.warn('[Gochi] WebSocket connection failed, continuing without real-time updates:', e);
        }

        // Start update timers
        startUpdateTimers();

        // Hide onboarding if visible
        GochiUI.hideOnboarding();
        GochiUI.hideLoading();

        console.log('[Gochi] Pet loaded:', pet.name);
    }

    // ========================================================================
    // Event Listeners
    // ========================================================================

    /**
     * Setup application event listeners
     */
    function setupEventListeners() {
        // Pet creation
        document.addEventListener('createPet', handleCreatePet);

        // Interactions
        document.addEventListener('interaction', handleInteraction);

        // Navigation
        document.addEventListener('navigation', handleNavigation);

        // Keyboard shortcuts
        document.addEventListener('keydown', handleKeyboard);
    }

    /**
     * Handle create pet event
     * @param {CustomEvent} e - Create pet event
     */
    async function handleCreatePet(e) {
        const { name, randomize } = e.detail;

        GochiUI.showLoading('Creating your pet...');

        try {
            const pet = await GochiAPI.pets.create(name, randomize);
            await selectPet(pet);

            GochiUI.toast.success(`Welcome ${name}!`);
            GochiPet.playEffect('levelUp');

        } catch (error) {
            console.error('[Gochi] Failed to create pet:', error);
            GochiUI.hideLoading();
            GochiUI.toast.error('Failed to create pet. Please try again.');
        }
    }

    /**
     * Handle interaction event
     * @param {CustomEvent} e - Interaction event
     */
    async function handleInteraction(e) {
        const { action } = e.detail;

        if (!currentPet) {
            GochiUI.toast.warning('No pet selected');
            return;
        }

        // Disable buttons during interaction
        GochiUI.setInteractionsEnabled(false);

        try {
            let result;

            switch (action) {
                case 'feed':
                    result = await GochiAPI.interactions.feed(currentPet.id);
                    GochiPet.playInteractionAnimation('feed');
                    break;

                case 'pet':
                    result = await GochiAPI.interactions.pet(currentPet.id);
                    GochiPet.playInteractionAnimation('pet');
                    break;

                case 'play':
                    result = await GochiAPI.interactions.play(currentPet.id);
                    GochiPet.playInteractionAnimation('play');
                    break;

                case 'groom':
                    result = await GochiAPI.interactions.groom(currentPet.id);
                    GochiPet.playInteractionAnimation('groom');
                    break;

                case 'train':
                    result = await GochiAPI.interactions.train(currentPet.id, 'obedience');
                    GochiPet.playInteractionAnimation('train');
                    break;

                case 'medical':
                    result = await GochiAPI.interactions.medical(currentPet.id);
                    GochiPet.playInteractionAnimation('medical');
                    break;

                default:
                    result = await GochiAPI.interactions.perform(currentPet.id, action);
                    GochiPet.playEffect('happy');
            }

            if (result) {
                // Show reaction message
                if (result.message) {
                    GochiPet.showMoodBubble(result.pet_reaction || result.message);
                }

                // Refresh pet status
                await refreshPetStatus();
            }

        } catch (error) {
            console.error('[Gochi] Interaction failed:', error);
            GochiUI.toast.error('Interaction failed');
        } finally {
            // Re-enable buttons after a delay
            setTimeout(() => {
                GochiUI.setInteractionsEnabled(true);
            }, 1000);
        }
    }

    /**
     * Handle navigation event
     * @param {CustomEvent} e - Navigation event
     */
    function handleNavigation(e) {
        const { view } = e.detail;

        // Load data for the view
        switch (view) {
            case 'social':
                GochiSocial.loadFriends();
                break;
            case 'achievements':
                loadAchievements();
                break;
        }
    }

    /**
     * Handle keyboard shortcuts
     * @param {KeyboardEvent} e - Keyboard event
     */
    function handleKeyboard(e) {
        // Only handle if not in input
        if (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA') {
            return;
        }

        switch (e.key) {
            case '1':
                GochiUI.navigateTo('pet');
                break;
            case '2':
                GochiUI.navigateTo('social');
                break;
            case '3':
                GochiUI.navigateTo('achievements');
                break;
            case 'f':
                if (!e.ctrlKey && !e.metaKey) {
                    document.dispatchEvent(new CustomEvent('interaction', { detail: { action: 'feed' } }));
                }
                break;
            case 'p':
                if (!e.ctrlKey && !e.metaKey) {
                    document.dispatchEvent(new CustomEvent('interaction', { detail: { action: 'play' } }));
                }
                break;
        }
    }

    // ========================================================================
    // Update Timers
    // ========================================================================

    /**
     * Start update timers
     */
    function startUpdateTimers() {
        stopUpdateTimers();

        // Pet simulation update
        updateTimer = setInterval(() => {
            if (currentPet) {
                triggerPetUpdate();
            }
        }, config.updateInterval);

        // Status polling
        statusTimer = setInterval(() => {
            if (currentPet) {
                refreshPetStatus();
            }
        }, config.statusPollInterval);

        // Auto-save
        autoSaveTimer = setInterval(() => {
            autoSave();
        }, config.autoSaveInterval);
    }

    /**
     * Stop update timers
     */
    function stopUpdateTimers() {
        if (updateTimer) {
            clearInterval(updateTimer);
            updateTimer = null;
        }
        if (statusTimer) {
            clearInterval(statusTimer);
            statusTimer = null;
        }
        if (autoSaveTimer) {
            clearInterval(autoSaveTimer);
            autoSaveTimer = null;
        }
    }

    /**
     * Trigger pet simulation update
     */
    async function triggerPetUpdate() {
        if (!currentPet) return;

        const now = Date.now();
        const deltaTime = (now - lastUpdateTime) / 1000; // seconds
        lastUpdateTime = now;

        try {
            await GochiAPI.pets.triggerUpdate(currentPet.id, deltaTime);
            await refreshPetStatus();
        } catch (error) {
            console.error('[Gochi] Pet update failed:', error);
        }
    }

    /**
     * Refresh pet status from server
     */
    async function refreshPetStatus() {
        if (!currentPet) return;

        try {
            const status = await GochiAPI.pets.getStatus(currentPet.id);

            // Update current pet data
            Object.assign(currentPet, status);

            // Update displays
            GochiPet.updatePetInfo(currentPet);
            GochiPet.setBehavior(status.current_behavior);

            // Check for critical needs
            if (status.critical_needs && status.critical_needs.length > 0) {
                handleCriticalNeeds(status.critical_needs);
            }

            // Get full vitals (optional)
            try {
                const vitals = await GochiAPI.pets.getVitals(currentPet.id);
                if (vitals) {
                    currentPet.vitals = vitals;
                    GochiPet.updateVitals(vitals);
                }
            } catch (e) {
                // Vitals might already be in status
                if (status.vitals) {
                    currentPet.vitals = status.vitals;
                    GochiPet.updateVitals(status.vitals);
                }
            }

            // Get emotions (optional)
            try {
                const emotions = await GochiAPI.pets.getEmotions(currentPet.id);
                if (emotions) {
                    currentPet.emotions = emotions;
                    GochiPet.updateEmotions(emotions);
                }
            } catch (e) {
                // Emotions might already be in status
                if (status.emotions) {
                    currentPet.emotions = status.emotions;
                    GochiPet.updateEmotions(status.emotions);
                }
            }

            GochiPet.updateStats(currentPet);

        } catch (error) {
            console.error('[Gochi] Status refresh failed:', error);
        }
    }

    /**
     * Handle critical needs notification
     * @param {string[]} needs - Array of critical need names
     */
    function handleCriticalNeeds(needs) {
        const needMessages = {
            health: 'Your pet needs medical attention!',
            energy: 'Your pet is exhausted!',
            nutrition: 'Your pet is hungry!',
            hydration: 'Your pet is thirsty!',
            happiness: 'Your pet is unhappy!',
            cleanliness: 'Your pet needs grooming!'
        };

        needs.forEach(need => {
            const message = needMessages[need.toLowerCase()] || `Your pet has critical ${need}!`;
            GochiUI.toast.warning(message);
        });
    }

    /**
     * Auto-save pet data
     */
    async function autoSave() {
        if (!currentPet) return;

        try {
            await GochiAPI.sync.sync(new Date(), 1, currentPet);
            console.log('[Gochi] Auto-save completed');
        } catch (error) {
            console.error('[Gochi] Auto-save failed:', error);
        }
    }

    // ========================================================================
    // WebSocket
    // ========================================================================

    /**
     * Connect to WebSocket
     */
    async function connectWebSocket() {
        try {
            await GochiWebSocket.connect(currentPet?.id);

            // Subscribe to updates
            if (currentPet) {
                GochiWebSocket.subscribeToPet(currentPet.id);
            }

            // Setup handlers
            GochiWebSocket.on('pet_update', handlePetUpdate);
            GochiWebSocket.on('vitals_update', handleVitalsUpdate);
            GochiWebSocket.on('emotion_update', handleEmotionUpdate);
            GochiWebSocket.on('achievement', handleAchievementUnlock);
            GochiWebSocket.on('notification', handleNotification);

            // Setup social WebSocket handlers
            GochiSocial.setupWebSocketHandlers();

        } catch (error) {
            console.error('[Gochi] WebSocket connection failed:', error);
            // Continue without WebSocket - polling will still work
        }
    }

    /**
     * Handle pet update from WebSocket
     * @param {object} data - Pet update data
     */
    function handlePetUpdate(data) {
        if (currentPet && data.id === currentPet.id) {
            Object.assign(currentPet, data);
            GochiPet.updatePetInfo(currentPet);
            GochiPet.setBehavior(data.current_behavior);
        }
    }

    /**
     * Handle vitals update from WebSocket
     * @param {object} data - Vitals data
     */
    function handleVitalsUpdate(data) {
        if (currentPet) {
            currentPet.vitals = data;
            GochiPet.updateVitals(data);
        }
    }

    /**
     * Handle emotion update from WebSocket
     * @param {object} data - Emotion data
     */
    function handleEmotionUpdate(data) {
        if (currentPet) {
            currentPet.emotions = data;
            GochiPet.updateEmotions(data);
        }
    }

    /**
     * Handle achievement unlock from WebSocket
     * @param {object} data - Achievement data
     */
    function handleAchievementUnlock(data) {
        GochiUI.toast.success(`Achievement Unlocked: ${data.name}!`);
        GochiPet.playEffect('levelUp');
    }

    /**
     * Handle notification from WebSocket
     * @param {object} data - Notification data
     */
    function handleNotification(data) {
        GochiUI.toast.info(data.message);
        updateNotificationCount();
    }

    // ========================================================================
    // Visibility Handling
    // ========================================================================

    /**
     * Setup visibility change handler
     */
    function setupVisibilityHandler() {
        document.addEventListener('visibilitychange', () => {
            if (document.visibilityState === 'visible') {
                // Tab became visible
                console.log('[Gochi] Tab visible, refreshing...');

                // Reconnect WebSocket if needed
                if (config.reconnectOnFocus && !GochiWebSocket.getIsConnected()) {
                    connectWebSocket();
                }

                // Refresh status
                refreshPetStatus();

            } else {
                // Tab hidden - could pause animations to save resources
            }
        });
    }

    // ========================================================================
    // Achievements
    // ========================================================================

    /**
     * Load achievements
     */
    async function loadAchievements() {
        try {
            const achievements = await GochiAPI.achievements.list();
            renderAchievements(achievements);
        } catch (error) {
            console.error('[Gochi] Failed to load achievements:', error);
        }
    }

    /**
     * Render achievements
     * @param {Array} achievements - Achievement data
     */
    function renderAchievements(achievements) {
        const grid = document.getElementById('achievements-grid');
        if (!grid) return;

        if (!achievements || achievements.length === 0) {
            grid.innerHTML = GochiUI.renderEmptyState('&#127942;', 'No achievements yet');
            return;
        }

        grid.innerHTML = achievements.map(a => GochiUI.renderAchievementCard(a)).join('');

        // Update counts
        const unlocked = achievements.filter(a => a.is_unlocked).length;
        const unlockedEl = document.getElementById('achievements-unlocked');
        const totalEl = document.getElementById('achievements-total');

        if (unlockedEl) unlockedEl.textContent = unlocked;
        if (totalEl) totalEl.textContent = achievements.length;
    }

    // ========================================================================
    // Notifications
    // ========================================================================

    /**
     * Update notification count
     */
    async function updateNotificationCount() {
        // This would fetch from API in production
        // For now, just update from friend requests
        const requests = GochiSocial.getFriends()?.length || 0;
        GochiUI.updateNotificationBadge(requests);
    }

    // ========================================================================
    // Public API
    // ========================================================================

    return {
        // Initialization
        init,

        // Pet management
        loadPet,
        selectPet,
        getCurrentPet: () => currentPet,
        refreshPetStatus,

        // User
        getCurrentUser: () => currentUser,

        // State
        isInitialized: () => isInitialized,

        // Configuration
        config
    };
})();

// ============================================================================
// Application Entry Point
// ============================================================================

// Initialize when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
    GochiApp.init();
});

// Export for module systems
if (typeof module !== 'undefined' && module.exports) {
    module.exports = GochiApp;
}
