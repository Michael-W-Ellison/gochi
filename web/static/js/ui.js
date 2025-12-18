/**
 * Gochi UI Management Module
 * Handles UI state, navigation, modals, toasts, and general UI utilities
 */

const GochiUI = (function() {
    'use strict';

    // ========================================================================
    // State
    // ========================================================================

    let currentView = 'pet';
    let currentSocialTab = 'friends';
    let currentAchievementCategory = 'all';
    let isLoading = false;

    // DOM Elements cache
    const elements = {};

    // ========================================================================
    // Initialization
    // ========================================================================

    /**
     * Initialize the UI module
     */
    function init() {
        cacheElements();
        setupNavigation();
        setupTabs();
        setupInteractionButtons();
        setupModal();
        setupOnboarding();

        // Hide loading overlay after init
        hideLoading();
    }

    /**
     * Cache DOM elements for performance
     */
    function cacheElements() {
        // Views
        elements.views = {
            pet: document.getElementById('pet-view'),
            social: document.getElementById('social-view'),
            achievements: document.getElementById('achievements-view')
        };

        // Navigation
        elements.navBtns = document.querySelectorAll('.nav-btn');

        // Social tabs
        elements.socialTabs = document.querySelectorAll('.tab-btn');
        elements.tabContents = document.querySelectorAll('.tab-content');

        // Modal
        elements.modalOverlay = document.getElementById('modal-overlay');
        elements.modal = document.getElementById('modal');
        elements.modalTitle = document.getElementById('modal-title');
        elements.modalContent = document.getElementById('modal-content');
        elements.modalFooter = document.getElementById('modal-footer');
        elements.modalClose = document.getElementById('modal-close');

        // Toast container
        elements.toastContainer = document.getElementById('toast-container');

        // Loading
        elements.loadingOverlay = document.getElementById('loading-overlay');

        // Onboarding
        elements.onboarding = document.getElementById('onboarding');
        elements.createPetForm = document.getElementById('create-pet-form');

        // Interaction buttons
        elements.interactBtns = document.querySelectorAll('.interact-btn');

        // Notification
        elements.notificationBadge = document.getElementById('notification-badge');
    }

    // ========================================================================
    // Navigation
    // ========================================================================

    /**
     * Setup navigation event listeners
     */
    function setupNavigation() {
        elements.navBtns.forEach(btn => {
            btn.addEventListener('click', () => {
                const view = btn.dataset.view;
                navigateTo(view);
            });
        });
    }

    /**
     * Navigate to a view
     * @param {string} viewName - View name to navigate to
     */
    function navigateTo(viewName) {
        if (!elements.views[viewName]) {
            console.warn('Unknown view:', viewName);
            return;
        }

        // Update nav buttons
        elements.navBtns.forEach(btn => {
            btn.classList.toggle('active', btn.dataset.view === viewName);
        });

        // Update views
        Object.entries(elements.views).forEach(([name, el]) => {
            if (el) {
                el.classList.toggle('active', name === viewName);
            }
        });

        currentView = viewName;

        // Emit navigation event
        document.dispatchEvent(new CustomEvent('navigation', { detail: { view: viewName } }));
    }

    /**
     * Get current view
     * @returns {string} Current view name
     */
    function getCurrentView() {
        return currentView;
    }

    // ========================================================================
    // Tabs
    // ========================================================================

    /**
     * Setup tab event listeners
     */
    function setupTabs() {
        elements.socialTabs.forEach(tab => {
            tab.addEventListener('click', () => {
                const tabName = tab.dataset.tab;
                switchTab(tabName);
            });
        });

        // Achievement categories
        const categoryBtns = document.querySelectorAll('.category-btn');
        categoryBtns.forEach(btn => {
            btn.addEventListener('click', () => {
                const category = btn.dataset.category;
                switchAchievementCategory(category);
            });
        });
    }

    /**
     * Switch to a social tab
     * @param {string} tabName - Tab name
     */
    function switchTab(tabName) {
        // Update tab buttons
        elements.socialTabs.forEach(tab => {
            tab.classList.toggle('active', tab.dataset.tab === tabName);
        });

        // Update tab contents
        elements.tabContents.forEach(content => {
            content.classList.toggle('active', content.id === `${tabName}-tab`);
        });

        currentSocialTab = tabName;

        // Emit tab change event
        document.dispatchEvent(new CustomEvent('tabChange', { detail: { tab: tabName } }));
    }

    /**
     * Switch achievement category
     * @param {string} category - Category name
     */
    function switchAchievementCategory(category) {
        const categoryBtns = document.querySelectorAll('.category-btn');
        categoryBtns.forEach(btn => {
            btn.classList.toggle('active', btn.dataset.category === category);
        });

        currentAchievementCategory = category;

        // Emit category change event
        document.dispatchEvent(new CustomEvent('achievementCategoryChange', { detail: { category } }));
    }

    // ========================================================================
    // Interaction Buttons
    // ========================================================================

    /**
     * Setup interaction button event listeners
     */
    function setupInteractionButtons() {
        elements.interactBtns.forEach(btn => {
            btn.addEventListener('click', () => {
                const action = btn.dataset.action;
                handleInteraction(action);
            });
        });
    }

    /**
     * Handle interaction button click
     * @param {string} action - Interaction action
     */
    function handleInteraction(action) {
        // Emit interaction event
        document.dispatchEvent(new CustomEvent('interaction', { detail: { action } }));
    }

    /**
     * Set interaction buttons enabled state
     * @param {boolean} enabled - Whether buttons should be enabled
     */
    function setInteractionsEnabled(enabled) {
        elements.interactBtns.forEach(btn => {
            btn.disabled = !enabled;
        });
    }

    // ========================================================================
    // Modal
    // ========================================================================

    /**
     * Setup modal event listeners
     */
    function setupModal() {
        if (elements.modalClose) {
            elements.modalClose.addEventListener('click', closeModal);
        }

        if (elements.modalOverlay) {
            elements.modalOverlay.addEventListener('click', (e) => {
                if (e.target === elements.modalOverlay) {
                    closeModal();
                }
            });
        }

        // Close on escape
        document.addEventListener('keydown', (e) => {
            if (e.key === 'Escape' && !elements.modalOverlay?.classList.contains('hidden')) {
                closeModal();
            }
        });
    }

    /**
     * Show modal dialog
     * @param {object} options - Modal options
     * @param {string} options.title - Modal title
     * @param {string|HTMLElement} options.content - Modal content
     * @param {Array} options.buttons - Modal buttons
     */
    function showModal({ title, content, buttons = [] }) {
        if (!elements.modalOverlay) return;

        // Set title
        if (elements.modalTitle) {
            elements.modalTitle.textContent = title;
        }

        // Set content
        if (elements.modalContent) {
            if (typeof content === 'string') {
                elements.modalContent.innerHTML = content;
            } else {
                elements.modalContent.innerHTML = '';
                elements.modalContent.appendChild(content);
            }
        }

        // Set buttons
        if (elements.modalFooter) {
            elements.modalFooter.innerHTML = '';

            buttons.forEach(({ text, type = 'secondary', onClick }) => {
                const btn = document.createElement('button');
                btn.className = `btn-${type}`;
                btn.textContent = text;
                btn.addEventListener('click', () => {
                    if (onClick) onClick();
                    closeModal();
                });
                elements.modalFooter.appendChild(btn);
            });
        }

        // Show modal
        elements.modalOverlay.classList.remove('hidden');
    }

    /**
     * Close modal dialog
     */
    function closeModal() {
        if (elements.modalOverlay) {
            elements.modalOverlay.classList.add('hidden');
        }
    }

    /**
     * Show confirmation dialog
     * @param {string} title - Dialog title
     * @param {string} message - Dialog message
     * @returns {Promise<boolean>} User confirmation
     */
    function confirm(title, message) {
        return new Promise((resolve) => {
            showModal({
                title,
                content: `<p>${message}</p>`,
                buttons: [
                    { text: 'Cancel', type: 'secondary', onClick: () => resolve(false) },
                    { text: 'Confirm', type: 'primary', onClick: () => resolve(true) }
                ]
            });
        });
    }

    /**
     * Show alert dialog
     * @param {string} title - Dialog title
     * @param {string} message - Dialog message
     */
    function alert(title, message) {
        showModal({
            title,
            content: `<p>${message}</p>`,
            buttons: [
                { text: 'OK', type: 'primary' }
            ]
        });
    }

    /**
     * Show prompt dialog
     * @param {string} title - Dialog title
     * @param {string} message - Dialog message
     * @param {string} defaultValue - Default input value
     * @returns {Promise<string|null>} User input or null if cancelled
     */
    function prompt(title, message, defaultValue = '') {
        return new Promise((resolve) => {
            const content = document.createElement('div');
            content.innerHTML = `
                <p>${message}</p>
                <div class="form-group">
                    <input type="text" id="prompt-input" value="${defaultValue}">
                </div>
            `;

            showModal({
                title,
                content,
                buttons: [
                    { text: 'Cancel', type: 'secondary', onClick: () => resolve(null) },
                    {
                        text: 'OK',
                        type: 'primary',
                        onClick: () => {
                            const input = document.getElementById('prompt-input');
                            resolve(input ? input.value : null);
                        }
                    }
                ]
            });

            // Focus input
            setTimeout(() => {
                const input = document.getElementById('prompt-input');
                if (input) input.focus();
            }, 100);
        });
    }

    // ========================================================================
    // Toast Notifications
    // ========================================================================

    /**
     * Show toast notification
     * @param {string} message - Toast message
     * @param {string} type - Toast type (success, error, warning, info)
     * @param {number} duration - Duration in milliseconds
     */
    function showToast(message, type = 'info', duration = 4000) {
        if (!elements.toastContainer) return;

        const toast = document.createElement('div');
        toast.className = `toast ${type}`;

        const icons = {
            success: '&#10003;',
            error: '&#10007;',
            warning: '&#9888;',
            info: '&#8505;'
        };

        toast.innerHTML = `
            <span class="toast-icon">${icons[type] || icons.info}</span>
            <span class="toast-message">${message}</span>
            <button class="toast-close">&times;</button>
        `;

        // Close button
        const closeBtn = toast.querySelector('.toast-close');
        closeBtn.addEventListener('click', () => removeToast(toast));

        elements.toastContainer.appendChild(toast);

        // Auto remove
        if (duration > 0) {
            setTimeout(() => removeToast(toast), duration);
        }

        return toast;
    }

    /**
     * Remove toast notification
     * @param {HTMLElement} toast - Toast element
     */
    function removeToast(toast) {
        toast.style.animation = 'toastSlideOut 0.3s ease forwards';
        setTimeout(() => toast.remove(), 300);
    }

    // Convenience methods
    const toast = {
        success: (message, duration) => showToast(message, 'success', duration),
        error: (message, duration) => showToast(message, 'error', duration),
        warning: (message, duration) => showToast(message, 'warning', duration),
        info: (message, duration) => showToast(message, 'info', duration)
    };

    // ========================================================================
    // Loading
    // ========================================================================

    /**
     * Show loading overlay
     * @param {string} message - Loading message
     */
    function showLoading(message = 'Loading...') {
        if (elements.loadingOverlay) {
            const text = elements.loadingOverlay.querySelector('p');
            if (text) text.textContent = message;
            elements.loadingOverlay.classList.remove('hidden');
        }
        isLoading = true;
    }

    /**
     * Hide loading overlay
     */
    function hideLoading() {
        if (elements.loadingOverlay) {
            elements.loadingOverlay.classList.add('hidden');
        }
        isLoading = false;
    }

    /**
     * Check if loading
     * @returns {boolean}
     */
    function getIsLoading() {
        return isLoading;
    }

    // ========================================================================
    // Onboarding
    // ========================================================================

    /**
     * Setup onboarding event listeners
     */
    function setupOnboarding() {
        if (elements.createPetForm) {
            elements.createPetForm.addEventListener('submit', handleCreatePet);
        }
    }

    /**
     * Show onboarding screen
     */
    function showOnboarding() {
        if (elements.onboarding) {
            elements.onboarding.classList.remove('hidden');
        }
    }

    /**
     * Hide onboarding screen
     */
    function hideOnboarding() {
        if (elements.onboarding) {
            elements.onboarding.classList.add('hidden');
        }
    }

    /**
     * Handle create pet form submission
     * @param {Event} e - Form event
     */
    function handleCreatePet(e) {
        e.preventDefault();

        const nameInput = document.getElementById('new-pet-name');
        const personalityInput = document.querySelector('input[name="personality"]:checked');

        const name = nameInput ? nameInput.value.trim() : '';
        const randomize = personalityInput ? personalityInput.value === 'random' : false;

        if (!name) {
            toast.error('Please enter a name for your pet');
            return;
        }

        // Emit create pet event
        document.dispatchEvent(new CustomEvent('createPet', {
            detail: { name, randomize }
        }));
    }

    // ========================================================================
    // Notifications
    // ========================================================================

    /**
     * Update notification badge
     * @param {number} count - Notification count
     */
    function updateNotificationBadge(count) {
        if (!elements.notificationBadge) return;

        if (count > 0) {
            elements.notificationBadge.textContent = count > 99 ? '99+' : count;
            elements.notificationBadge.classList.remove('hidden');
        } else {
            elements.notificationBadge.classList.add('hidden');
        }
    }

    // ========================================================================
    // List Rendering Helpers
    // ========================================================================

    /**
     * Render empty state
     * @param {string} icon - Icon emoji
     * @param {string} message - Empty state message
     * @returns {string} HTML string
     */
    function renderEmptyState(icon, message) {
        return `
            <div class="empty-state">
                <span class="empty-icon">${icon}</span>
                <p>${message}</p>
            </div>
        `;
    }

    /**
     * Render friend card
     * @param {object} friend - Friend data
     * @returns {string} HTML string
     */
    function renderFriendCard(friend) {
        return `
            <div class="friend-card" data-pet-id="${friend.pet_id}">
                <div class="friend-avatar">${friend.name.charAt(0)}</div>
                <div class="friend-info">
                    <span class="friend-name">${friend.name}</span>
                    <span class="friend-status">${friend.is_online ? 'Online' : 'Offline'}</span>
                </div>
                <div class="friend-actions">
                    <button class="btn-small btn-secondary" data-action="visit">Visit</button>
                    <button class="btn-small btn-secondary" data-action="wave">Wave</button>
                </div>
            </div>
        `;
    }

    /**
     * Render nearby pet card
     * @param {object} pet - Nearby pet data
     * @returns {string} HTML string
     */
    function renderNearbyCard(pet) {
        return `
            <div class="nearby-card" data-pet-id="${pet.pet_id}">
                <div class="nearby-avatar">${pet.name.charAt(0)}</div>
                <div class="nearby-info">
                    <span class="nearby-name">${pet.name}</span>
                    <span class="nearby-distance">${formatDistance(pet.distance_meters)} ${pet.direction}</span>
                </div>
                <div class="nearby-actions">
                    <button class="btn-small btn-primary" data-action="add-friend">Add Friend</button>
                </div>
            </div>
        `;
    }

    /**
     * Render feed item
     * @param {object} item - Feed item data
     * @returns {string} HTML string
     */
    function renderFeedItem(item) {
        return `
            <div class="feed-item" data-item-id="${item.id}">
                <div class="feed-item-header">
                    <div class="feed-item-avatar">${item.pet_name.charAt(0)}</div>
                    <div class="feed-item-meta">
                        <span class="feed-item-author">${item.pet_name}</span>
                        <span class="feed-item-time">${formatTime(item.created_at)}</span>
                    </div>
                </div>
                <div class="feed-item-content">${item.content}</div>
                ${item.image_url ? `<img class="feed-item-image" src="${item.image_url}" alt="">` : ''}
                <div class="feed-item-actions">
                    <button class="feed-action-btn ${item.has_liked ? 'liked' : ''}" data-action="like">
                        &#10084; ${item.like_count}
                    </button>
                </div>
            </div>
        `;
    }

    /**
     * Render event card
     * @param {object} event - Event data
     * @returns {string} HTML string
     */
    function renderEventCard(event) {
        const date = new Date(event.start_time);
        return `
            <div class="event-card" data-event-id="${event.id}">
                <div class="event-date">
                    <span class="event-day">${date.getDate()}</span>
                    <span class="event-month">${date.toLocaleDateString('en', { month: 'short' })}</span>
                </div>
                <div class="event-info">
                    <span class="event-title">${event.name}</span>
                    <span class="event-description">${event.description}</span>
                    <div class="event-meta">
                        <span>&#128101; ${event.current_participants}/${event.max_participants}</span>
                        <span>&#128205; ${event.location?.name || 'Online'}</span>
                    </div>
                </div>
                <button class="btn-small ${event.is_joined ? 'btn-secondary' : 'btn-primary'}" data-action="${event.is_joined ? 'leave' : 'join'}">
                    ${event.is_joined ? 'Leave' : 'Join'}
                </button>
            </div>
        `;
    }

    /**
     * Render achievement card
     * @param {object} achievement - Achievement data
     * @returns {string} HTML string
     */
    function renderAchievementCard(achievement) {
        return `
            <div class="achievement-card ${achievement.is_unlocked ? '' : 'locked'}" data-achievement-id="${achievement.id}">
                <div class="achievement-icon">${achievement.icon_url || '&#127942;'}</div>
                <div class="achievement-info">
                    <span class="achievement-name">${achievement.name}</span>
                    <span class="achievement-description">${achievement.description}</span>
                    ${!achievement.is_unlocked ? `
                        <div class="achievement-progress-bar">
                            <div class="fill" style="width: ${achievement.progress * 100}%"></div>
                        </div>
                    ` : ''}
                </div>
            </div>
        `;
    }

    // ========================================================================
    // Formatting Helpers
    // ========================================================================

    /**
     * Format distance
     * @param {number} meters - Distance in meters
     * @returns {string} Formatted distance
     */
    function formatDistance(meters) {
        if (meters < 1000) {
            return `${Math.round(meters)}m`;
        }
        return `${(meters / 1000).toFixed(1)}km`;
    }

    /**
     * Format relative time
     * @param {string} timestamp - ISO timestamp
     * @returns {string} Relative time string
     */
    function formatTime(timestamp) {
        const date = new Date(timestamp);
        const now = new Date();
        const diff = now - date;

        const minutes = Math.floor(diff / 60000);
        const hours = Math.floor(diff / 3600000);
        const days = Math.floor(diff / 86400000);

        if (minutes < 1) return 'Just now';
        if (minutes < 60) return `${minutes}m ago`;
        if (hours < 24) return `${hours}h ago`;
        if (days < 7) return `${days}d ago`;

        return date.toLocaleDateString();
    }

    // ========================================================================
    // Public API
    // ========================================================================

    return {
        // Initialization
        init,

        // Navigation
        navigateTo,
        getCurrentView,
        switchTab,
        switchAchievementCategory,

        // Interactions
        setInteractionsEnabled,

        // Modal
        showModal,
        closeModal,
        confirm,
        alert,
        prompt,

        // Toast
        showToast,
        toast,

        // Loading
        showLoading,
        hideLoading,
        getIsLoading,

        // Onboarding
        showOnboarding,
        hideOnboarding,

        // Notifications
        updateNotificationBadge,

        // Rendering helpers
        renderEmptyState,
        renderFriendCard,
        renderNearbyCard,
        renderFeedItem,
        renderEventCard,
        renderAchievementCard,

        // Formatting helpers
        formatDistance,
        formatTime
    };
})();

// Export for module systems
if (typeof module !== 'undefined' && module.exports) {
    module.exports = GochiUI;
}
