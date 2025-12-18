/**
 * Gochi Social Features Module
 * Handles friends, nearby pets, social feed, and events
 */

const GochiSocial = (function() {
    'use strict';

    // ========================================================================
    // State
    // ========================================================================

    let currentPetId = null;
    let friends = [];
    let nearbyPets = [];
    let feedItems = [];
    let events = [];
    let friendRequests = [];
    let userLocation = null;
    let locationWatchId = null;

    // ========================================================================
    // Initialization
    // ========================================================================

    /**
     * Initialize social module
     * @param {string} petId - Current pet ID
     */
    function init(petId) {
        currentPetId = petId;
        setupEventListeners();
    }

    /**
     * Setup event listeners
     */
    function setupEventListeners() {
        // Tab changes
        document.addEventListener('tabChange', handleTabChange);

        // Friend list actions
        document.getElementById('friends-list')?.addEventListener('click', handleFriendAction);

        // Nearby list actions
        document.getElementById('nearby-list')?.addEventListener('click', handleNearbyAction);

        // Refresh nearby button
        document.getElementById('refresh-nearby-btn')?.addEventListener('click', refreshNearbyPets);

        // Add friend button
        document.getElementById('add-friend-btn')?.addEventListener('click', showAddFriendDialog);

        // Feed actions
        document.getElementById('feed-list')?.addEventListener('click', handleFeedAction);

        // New post button
        document.getElementById('new-post-btn')?.addEventListener('click', showNewPostDialog);

        // Events filter
        document.getElementById('events-filter')?.addEventListener('change', handleEventsFilterChange);

        // Events list actions
        document.getElementById('events-list')?.addEventListener('click', handleEventAction);

        // Friend requests
        document.getElementById('requests-list')?.addEventListener('click', handleFriendRequestAction);
    }

    // ========================================================================
    // Tab Handling
    // ========================================================================

    /**
     * Handle tab change
     * @param {CustomEvent} e - Tab change event
     */
    function handleTabChange(e) {
        const { tab } = e.detail;

        switch (tab) {
            case 'friends':
                loadFriends();
                loadFriendRequests();
                break;
            case 'nearby':
                loadNearbyPets();
                break;
            case 'feed':
                loadFeed();
                break;
            case 'events':
                loadEvents();
                break;
        }
    }

    // ========================================================================
    // Friends
    // ========================================================================

    /**
     * Load friends list
     */
    async function loadFriends() {
        if (!currentPetId) return;

        try {
            friends = await GochiAPI.social.getFriends(currentPetId);
            renderFriends();
        } catch (error) {
            console.error('Failed to load friends:', error);
            GochiUI.toast.error('Failed to load friends');
        }
    }

    /**
     * Render friends list
     */
    function renderFriends() {
        const container = document.getElementById('friends-list');
        if (!container) return;

        if (!friends || friends.length === 0) {
            container.innerHTML = GochiUI.renderEmptyState('&#128101;', 'No friends yet. Find nearby pets to make friends!');
            return;
        }

        container.innerHTML = friends.map(friend => GochiUI.renderFriendCard(friend)).join('');
    }

    /**
     * Load friend requests
     */
    async function loadFriendRequests() {
        if (!currentPetId) return;

        try {
            friendRequests = await GochiAPI.social.getFriendRequests(currentPetId);
            renderFriendRequests();
        } catch (error) {
            console.error('Failed to load friend requests:', error);
        }
    }

    /**
     * Render friend requests
     */
    function renderFriendRequests() {
        const container = document.getElementById('requests-list');
        const section = document.getElementById('friend-requests');

        if (!container || !section) return;

        if (!friendRequests || friendRequests.length === 0) {
            section.classList.add('hidden');
            return;
        }

        section.classList.remove('hidden');

        container.innerHTML = friendRequests.map(request => `
            <div class="friend-card" data-request-id="${request.id}">
                <div class="friend-avatar">${request.from_pet_name.charAt(0)}</div>
                <div class="friend-info">
                    <span class="friend-name">${request.from_pet_name}</span>
                    <span class="friend-status">${request.message || 'Wants to be friends!'}</span>
                </div>
                <div class="friend-actions">
                    <button class="btn-small btn-primary" data-action="accept">Accept</button>
                    <button class="btn-small btn-secondary" data-action="decline">Decline</button>
                </div>
            </div>
        `).join('');
    }

    /**
     * Handle friend action clicks
     * @param {Event} e - Click event
     */
    async function handleFriendAction(e) {
        const btn = e.target.closest('button[data-action]');
        if (!btn) return;

        const card = btn.closest('.friend-card');
        const petId = card?.dataset.petId;
        const action = btn.dataset.action;

        if (!petId) return;

        try {
            switch (action) {
                case 'visit':
                    await visitFriend(petId);
                    break;
                case 'wave':
                    await waveAtFriend(petId);
                    break;
                case 'remove':
                    await removeFriend(petId);
                    break;
            }
        } catch (error) {
            console.error('Friend action failed:', error);
            GochiUI.toast.error('Action failed');
        }
    }

    /**
     * Handle friend request actions
     * @param {Event} e - Click event
     */
    async function handleFriendRequestAction(e) {
        const btn = e.target.closest('button[data-action]');
        if (!btn) return;

        const card = btn.closest('.friend-card');
        const requestId = card?.dataset.requestId;
        const action = btn.dataset.action;

        if (!requestId) return;

        try {
            if (action === 'accept') {
                await GochiAPI.social.acceptFriendRequest(requestId);
                GochiUI.toast.success('Friend request accepted!');
            } else if (action === 'decline') {
                await GochiAPI.social.declineFriendRequest(requestId);
                GochiUI.toast.info('Friend request declined');
            }

            loadFriendRequests();
            loadFriends();
        } catch (error) {
            console.error('Friend request action failed:', error);
            GochiUI.toast.error('Action failed');
        }
    }

    /**
     * Visit a friend's pet
     * @param {string} petId - Friend's pet ID
     */
    async function visitFriend(petId) {
        try {
            await GochiAPI.social.interact(currentPetId, petId, 'visit');
            GochiUI.toast.success('Visiting friend!');
        } catch (error) {
            throw error;
        }
    }

    /**
     * Wave at a friend
     * @param {string} petId - Friend's pet ID
     */
    async function waveAtFriend(petId) {
        try {
            await GochiAPI.social.interact(currentPetId, petId, 'wave');
            GochiUI.toast.success('Waved at friend!');
        } catch (error) {
            throw error;
        }
    }

    /**
     * Remove a friend
     * @param {string} petId - Friend's pet ID
     */
    async function removeFriend(petId) {
        const confirmed = await GochiUI.confirm('Remove Friend', 'Are you sure you want to remove this friend?');
        if (!confirmed) return;

        try {
            await GochiAPI.social.removeFriend(currentPetId, petId);
            GochiUI.toast.success('Friend removed');
            loadFriends();
        } catch (error) {
            throw error;
        }
    }

    /**
     * Show add friend dialog
     */
    async function showAddFriendDialog() {
        const petName = await GochiUI.prompt('Add Friend', 'Enter pet name to search:');
        if (!petName) return;

        try {
            const results = await GochiAPI.social.searchPets(petName);
            if (results && results.length > 0) {
                showSearchResults(results);
            } else {
                GochiUI.toast.info('No pets found');
            }
        } catch (error) {
            console.error('Search failed:', error);
            GochiUI.toast.error('Search failed');
        }
    }

    /**
     * Show search results in modal
     * @param {Array} results - Search results
     */
    function showSearchResults(results) {
        const content = document.createElement('div');
        content.innerHTML = results.map(pet => `
            <div class="friend-card" data-pet-id="${pet.pet_id}">
                <div class="friend-avatar">${pet.name.charAt(0)}</div>
                <div class="friend-info">
                    <span class="friend-name">${pet.name}</span>
                    <span class="friend-status">${pet.owner_name}</span>
                </div>
                <button class="btn-small btn-primary add-friend-btn">Add</button>
            </div>
        `).join('');

        content.addEventListener('click', async (e) => {
            const btn = e.target.closest('.add-friend-btn');
            if (!btn) return;

            const card = btn.closest('.friend-card');
            const petId = card?.dataset.petId;

            if (petId) {
                try {
                    await GochiAPI.social.sendFriendRequest(currentPetId, petId);
                    GochiUI.toast.success('Friend request sent!');
                    GochiUI.closeModal();
                } catch (error) {
                    GochiUI.toast.error('Failed to send request');
                }
            }
        });

        GochiUI.showModal({
            title: 'Search Results',
            content,
            buttons: [{ text: 'Close', type: 'secondary' }]
        });
    }

    // ========================================================================
    // Nearby Pets
    // ========================================================================

    /**
     * Load nearby pets
     */
    async function loadNearbyPets() {
        // Request location if not available
        if (!userLocation) {
            requestLocation();
            return;
        }

        try {
            nearbyPets = await GochiAPI.social.getNearbyPets(
                userLocation.latitude,
                userLocation.longitude,
                1000 // 1km radius
            );
            renderNearbyPets();
        } catch (error) {
            console.error('Failed to load nearby pets:', error);
            GochiUI.toast.error('Failed to load nearby pets');
        }
    }

    /**
     * Render nearby pets list
     */
    function renderNearbyPets() {
        const container = document.getElementById('nearby-list');
        if (!container) return;

        if (!nearbyPets || nearbyPets.length === 0) {
            container.innerHTML = GochiUI.renderEmptyState('&#128269;', 'No pets nearby. Try expanding your search radius!');
            return;
        }

        container.innerHTML = nearbyPets.map(pet => GochiUI.renderNearbyCard(pet)).join('');
    }

    /**
     * Refresh nearby pets
     */
    function refreshNearbyPets() {
        updateLocationStatus('Searching...');
        requestLocation();
    }

    /**
     * Handle nearby pet actions
     * @param {Event} e - Click event
     */
    async function handleNearbyAction(e) {
        const btn = e.target.closest('button[data-action]');
        if (!btn) return;

        const card = btn.closest('.nearby-card');
        const petId = card?.dataset.petId;
        const action = btn.dataset.action;

        if (!petId) return;

        try {
            if (action === 'add-friend') {
                await GochiAPI.social.sendFriendRequest(currentPetId, petId);
                GochiUI.toast.success('Friend request sent!');
            }
        } catch (error) {
            console.error('Nearby action failed:', error);
            GochiUI.toast.error('Action failed');
        }
    }

    /**
     * Request user location
     */
    function requestLocation() {
        if (!navigator.geolocation) {
            updateLocationStatus('Location not supported');
            return;
        }

        updateLocationStatus('Getting location...');

        navigator.geolocation.getCurrentPosition(
            (position) => {
                userLocation = {
                    latitude: position.coords.latitude,
                    longitude: position.coords.longitude
                };
                updateLocationStatus(`Location: ${userLocation.latitude.toFixed(4)}, ${userLocation.longitude.toFixed(4)}`);
                loadNearbyPets();

                // Update location on server
                GochiAPI.social.updateLocation(userLocation.latitude, userLocation.longitude);
            },
            (error) => {
                console.error('Location error:', error);
                updateLocationStatus('Location access denied');
            },
            { enableHighAccuracy: true, timeout: 10000 }
        );
    }

    /**
     * Update location status display
     * @param {string} text - Status text
     */
    function updateLocationStatus(text) {
        const statusEl = document.getElementById('location-status');
        if (statusEl) {
            const textEl = statusEl.querySelector('.status-text');
            if (textEl) textEl.textContent = text;
        }
    }

    // ========================================================================
    // Social Feed
    // ========================================================================

    /**
     * Load social feed
     */
    async function loadFeed() {
        try {
            const response = await GochiAPI.feed.get(1, 20);
            feedItems = response.items || response || [];
            renderFeed();
        } catch (error) {
            console.error('Failed to load feed:', error);
            GochiUI.toast.error('Failed to load feed');
        }
    }

    /**
     * Render social feed
     */
    function renderFeed() {
        const container = document.getElementById('feed-list');
        if (!container) return;

        if (!feedItems || feedItems.length === 0) {
            container.innerHTML = GochiUI.renderEmptyState('&#128240;', "No posts yet. Share your pet's adventures!");
            return;
        }

        container.innerHTML = feedItems.map(item => GochiUI.renderFeedItem(item)).join('');
    }

    /**
     * Handle feed actions
     * @param {Event} e - Click event
     */
    async function handleFeedAction(e) {
        const btn = e.target.closest('button[data-action]');
        if (!btn) return;

        const item = btn.closest('.feed-item');
        const itemId = item?.dataset.itemId;
        const action = btn.dataset.action;

        if (!itemId) return;

        try {
            if (action === 'like') {
                const isLiked = btn.classList.contains('liked');
                if (isLiked) {
                    await GochiAPI.feed.unlike(itemId);
                } else {
                    await GochiAPI.feed.like(itemId);
                }
                loadFeed(); // Refresh to update like count
            }
        } catch (error) {
            console.error('Feed action failed:', error);
            GochiUI.toast.error('Action failed');
        }
    }

    /**
     * Show new post dialog
     */
    async function showNewPostDialog() {
        const content = document.createElement('div');
        content.innerHTML = `
            <div class="form-group">
                <label for="post-content">What's happening?</label>
                <textarea id="post-content" rows="4" placeholder="Share an update about your pet..."></textarea>
            </div>
        `;

        GochiUI.showModal({
            title: 'New Post',
            content,
            buttons: [
                { text: 'Cancel', type: 'secondary' },
                {
                    text: 'Post',
                    type: 'primary',
                    onClick: async () => {
                        const textarea = document.getElementById('post-content');
                        const postContent = textarea?.value.trim();

                        if (!postContent) {
                            GochiUI.toast.warning('Please enter some content');
                            return;
                        }

                        try {
                            await GochiAPI.feed.post(postContent);
                            GochiUI.toast.success('Posted!');
                            loadFeed();
                        } catch (error) {
                            GochiUI.toast.error('Failed to post');
                        }
                    }
                }
            ]
        });
    }

    // ========================================================================
    // Events
    // ========================================================================

    /**
     * Load events
     * @param {string} filter - Filter type
     */
    async function loadEvents(filter = 'all') {
        try {
            events = await GochiAPI.events.list(filter);
            renderEvents();
        } catch (error) {
            console.error('Failed to load events:', error);
            GochiUI.toast.error('Failed to load events');
        }
    }

    /**
     * Render events list
     */
    function renderEvents() {
        const container = document.getElementById('events-list');
        if (!container) return;

        if (!events || events.length === 0) {
            container.innerHTML = GochiUI.renderEmptyState('&#127881;', 'No active events right now');
            return;
        }

        container.innerHTML = events.map(event => GochiUI.renderEventCard(event)).join('');
    }

    /**
     * Handle events filter change
     * @param {Event} e - Change event
     */
    function handleEventsFilterChange(e) {
        const filter = e.target.value;
        loadEvents(filter);
    }

    /**
     * Handle event actions
     * @param {Event} e - Click event
     */
    async function handleEventAction(e) {
        const btn = e.target.closest('button[data-action]');
        if (!btn) return;

        const card = btn.closest('.event-card');
        const eventId = card?.dataset.eventId;
        const action = btn.dataset.action;

        if (!eventId) return;

        try {
            if (action === 'join') {
                await GochiAPI.events.join(eventId);
                GochiUI.toast.success('Joined event!');
            } else if (action === 'leave') {
                await GochiAPI.events.leave(eventId);
                GochiUI.toast.info('Left event');
            }
            loadEvents();
        } catch (error) {
            console.error('Event action failed:', error);
            GochiUI.toast.error('Action failed');
        }
    }

    // ========================================================================
    // WebSocket Integration
    // ========================================================================

    /**
     * Setup WebSocket event handlers
     */
    function setupWebSocketHandlers() {
        GochiWebSocket.on('nearby_pet', handleNearbyPetUpdate);
        GochiWebSocket.on('friend_request', handleFriendRequestUpdate);
        GochiWebSocket.on('social_interaction', handleSocialInteractionUpdate);
        GochiWebSocket.on('feed_item', handleFeedItemUpdate);
        GochiWebSocket.on('event_update', handleEventUpdate);
    }

    /**
     * Handle nearby pet WebSocket update
     * @param {object} data - Update data
     */
    function handleNearbyPetUpdate(data) {
        // Add or update nearby pet
        const index = nearbyPets.findIndex(p => p.pet_id === data.pet_id);
        if (index >= 0) {
            nearbyPets[index] = data;
        } else {
            nearbyPets.push(data);
        }
        renderNearbyPets();
    }

    /**
     * Handle friend request WebSocket update
     * @param {object} data - Update data
     */
    function handleFriendRequestUpdate(data) {
        friendRequests.push(data);
        renderFriendRequests();
        GochiUI.toast.info(`${data.from_pet_name} wants to be friends!`);
    }

    /**
     * Handle social interaction WebSocket update
     * @param {object} data - Update data
     */
    function handleSocialInteractionUpdate(data) {
        GochiUI.toast.info(`${data.from_pet_name} ${data.type}d at you!`);
    }

    /**
     * Handle feed item WebSocket update
     * @param {object} data - Update data
     */
    function handleFeedItemUpdate(data) {
        feedItems.unshift(data);
        renderFeed();
    }

    /**
     * Handle event WebSocket update
     * @param {object} data - Update data
     */
    function handleEventUpdate(data) {
        const index = events.findIndex(e => e.id === data.id);
        if (index >= 0) {
            events[index] = data;
        } else {
            events.unshift(data);
        }
        renderEvents();
    }

    // ========================================================================
    // Public API
    // ========================================================================

    return {
        // Initialization
        init,
        setupWebSocketHandlers,

        // Friends
        loadFriends,
        loadFriendRequests,

        // Nearby
        loadNearbyPets,
        requestLocation,

        // Feed
        loadFeed,

        // Events
        loadEvents,

        // Getters
        getFriends: () => friends,
        getNearbyPets: () => nearbyPets,
        getFeedItems: () => feedItems,
        getEvents: () => events,
        getUserLocation: () => userLocation
    };
})();

// Export for module systems
if (typeof module !== 'undefined' && module.exports) {
    module.exports = GochiSocial;
}
