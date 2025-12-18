/**
 * Gochi API Client
 * Handles all HTTP communication with the backend API
 */

const GochiAPI = (function() {
    'use strict';

    // Configuration
    const config = {
        baseURL: '/api/v1',
        timeout: 30000,
        retryAttempts: 3,
        retryDelay: 1000
    };

    // State
    let authToken = null;
    let refreshToken = null;

    // ========================================================================
    // HTTP Client
    // ========================================================================

    /**
     * Make an HTTP request
     * @param {string} method - HTTP method
     * @param {string} endpoint - API endpoint
     * @param {object} data - Request body data
     * @param {object} options - Additional options
     * @returns {Promise<object>} Response data
     */
    async function request(method, endpoint, data = null, options = {}) {
        const url = `${config.baseURL}${endpoint}`;

        const headers = {
            'Content-Type': 'application/json',
            ...options.headers
        };

        if (authToken) {
            headers['Authorization'] = `Bearer ${authToken}`;
        }

        const fetchOptions = {
            method,
            headers,
            ...options
        };

        if (data && method !== 'GET') {
            fetchOptions.body = JSON.stringify(data);
        }

        try {
            const controller = new AbortController();
            const timeoutId = setTimeout(() => controller.abort(), config.timeout);
            fetchOptions.signal = controller.signal;

            const response = await fetch(url, fetchOptions);
            clearTimeout(timeoutId);

            // Handle token refresh
            if (response.status === 401 && refreshToken) {
                const refreshed = await refreshAuthToken();
                if (refreshed) {
                    headers['Authorization'] = `Bearer ${authToken}`;
                    return request(method, endpoint, data, options);
                }
            }

            const result = await response.json();

            if (!response.ok) {
                throw new APIError(
                    result.error?.message || 'Request failed',
                    result.error?.code || 'UNKNOWN_ERROR',
                    response.status
                );
            }

            return result.data;
        } catch (error) {
            if (error.name === 'AbortError') {
                throw new APIError('Request timeout', 'TIMEOUT', 408);
            }
            if (error instanceof APIError) {
                throw error;
            }
            throw new APIError(error.message, 'NETWORK_ERROR', 0);
        }
    }

    // HTTP method shortcuts
    const get = (endpoint, options) => request('GET', endpoint, null, options);
    const post = (endpoint, data, options) => request('POST', endpoint, data, options);
    const put = (endpoint, data, options) => request('PUT', endpoint, data, options);
    const del = (endpoint, options) => request('DELETE', endpoint, null, options);

    // ========================================================================
    // Error Handling
    // ========================================================================

    class APIError extends Error {
        constructor(message, code, status) {
            super(message);
            this.name = 'APIError';
            this.code = code;
            this.status = status;
        }
    }

    // ========================================================================
    // Authentication
    // ========================================================================

    const auth = {
        /**
         * Register a new user
         * @param {string} email - User email
         * @param {string} password - User password
         * @param {string} displayName - Display name
         */
        async register(email, password, displayName) {
            const data = await post('/auth/register', { email, password, display_name: displayName });
            setTokens(data.access_token, data.refresh_token);
            return data;
        },

        /**
         * Login user
         * @param {string} email - User email
         * @param {string} password - User password
         */
        async login(email, password) {
            const data = await post('/auth/login', { email, password });
            setTokens(data.access_token, data.refresh_token);
            return data;
        },

        /**
         * Logout user
         */
        async logout() {
            try {
                await post('/auth/logout');
            } finally {
                clearTokens();
            }
        },

        /**
         * Get current user profile
         */
        async getProfile() {
            return get('/auth/me');
        },

        /**
         * Update user profile
         * @param {object} updates - Profile updates
         */
        async updateProfile(updates) {
            return put('/auth/me', updates);
        },

        /**
         * Check if user is authenticated
         */
        isAuthenticated() {
            return !!authToken;
        }
    };

    function setTokens(access, refresh) {
        authToken = access;
        refreshToken = refresh;
        localStorage.setItem('gochi_auth_token', access);
        localStorage.setItem('gochi_refresh_token', refresh);
    }

    function clearTokens() {
        authToken = null;
        refreshToken = null;
        localStorage.removeItem('gochi_auth_token');
        localStorage.removeItem('gochi_refresh_token');
    }

    function loadTokens() {
        authToken = localStorage.getItem('gochi_auth_token');
        refreshToken = localStorage.getItem('gochi_refresh_token');
    }

    async function refreshAuthToken() {
        try {
            const data = await post('/auth/refresh', { refresh_token: refreshToken });
            setTokens(data.access_token, data.refresh_token);
            return true;
        } catch (error) {
            clearTokens();
            return false;
        }
    }

    // ========================================================================
    // Pet API
    // ========================================================================

    const pets = {
        /**
         * Create a new pet
         * @param {string} name - Pet name
         * @param {boolean} randomizeTraits - Whether to randomize personality
         */
        async create(name, randomizeTraits = false) {
            return post('/pets', { name, randomize_traits: randomizeTraits });
        },

        /**
         * Get all pets for current user
         */
        async list() {
            return get('/pets');
        },

        /**
         * Get a specific pet by ID
         * @param {string} petId - Pet ID
         */
        async get(petId) {
            return get(`/pets/${petId}`);
        },

        /**
         * Update pet properties
         * @param {string} petId - Pet ID
         * @param {object} updates - Updates to apply
         */
        async update(petId, updates) {
            return put(`/pets/${petId}`, updates);
        },

        /**
         * Delete a pet
         * @param {string} petId - Pet ID
         */
        async delete(petId) {
            return del(`/pets/${petId}`);
        },

        /**
         * Get pet status (lightweight)
         * @param {string} petId - Pet ID
         */
        async getStatus(petId) {
            return get(`/pets/${petId}/status`);
        },

        /**
         * Get pet vitals
         * @param {string} petId - Pet ID
         */
        async getVitals(petId) {
            return get(`/pets/${petId}/vitals`);
        },

        /**
         * Get pet emotions
         * @param {string} petId - Pet ID
         */
        async getEmotions(petId) {
            return get(`/pets/${petId}/emotions`);
        },

        /**
         * Get pet personality
         * @param {string} petId - Pet ID
         */
        async getPersonality(petId) {
            return get(`/pets/${petId}/personality`);
        },

        /**
         * Trigger simulation update
         * @param {string} petId - Pet ID
         * @param {number} deltaTime - Time delta in seconds
         */
        async triggerUpdate(petId, deltaTime) {
            return post(`/pets/${petId}/update`, { delta_time: deltaTime });
        }
    };

    // ========================================================================
    // Interactions API
    // ========================================================================

    const interactions = {
        /**
         * Perform a generic interaction
         * @param {string} petId - Pet ID
         * @param {string} type - Interaction type
         * @param {number} intensity - Interaction intensity (0-1)
         */
        async perform(petId, type, intensity = 0.5) {
            return post(`/pets/${petId}/interactions`, { type, intensity });
        },

        /**
         * Feed the pet
         * @param {string} petId - Pet ID
         * @param {string} foodType - Type of food
         * @param {number} quantity - Amount (0-1)
         */
        async feed(petId, foodType = 'basic', quantity = 0.5) {
            return post(`/pets/${petId}/interactions/feed`, { food_type: foodType, quantity });
        },

        /**
         * Pet/stroke the pet
         * @param {string} petId - Pet ID
         * @param {number} duration - Duration in seconds
         */
        async pet(petId, duration = 5) {
            return post(`/pets/${petId}/interactions/pet`, { duration });
        },

        /**
         * Play with the pet
         * @param {string} petId - Pet ID
         * @param {number} enthusiasm - Enthusiasm level (0-1)
         * @param {number} duration - Duration in seconds
         */
        async play(petId, enthusiasm = 0.7, duration = 10) {
            return post(`/pets/${petId}/interactions/play`, { enthusiasm, duration });
        },

        /**
         * Train the pet
         * @param {string} petId - Pet ID
         * @param {string} skillType - Skill to train
         * @param {string} method - Training method
         */
        async train(petId, skillType, method = 'positive_reinforcement') {
            return post(`/pets/${petId}/interactions/train`, { skill_type: skillType, method });
        },

        /**
         * Groom the pet
         * @param {string} petId - Pet ID
         * @param {string} groomType - Type of grooming
         * @param {number} duration - Duration in seconds
         */
        async groom(petId, groomType = 'brush', duration = 5) {
            return post(`/pets/${petId}/interactions/groom`, { groom_type: groomType, duration });
        },

        /**
         * Provide medical care
         * @param {string} petId - Pet ID
         */
        async medical(petId) {
            return post(`/pets/${petId}/interactions/medical`, {});
        },

        /**
         * Give reward
         * @param {string} petId - Pet ID
         */
        async reward(petId) {
            return post(`/pets/${petId}/interactions/reward`, {});
        },

        /**
         * Get interaction history
         * @param {string} petId - Pet ID
         * @param {number} limit - Max items to return
         */
        async getHistory(petId, limit = 20) {
            return get(`/pets/${petId}/interactions/history?limit=${limit}`);
        }
    };

    // ========================================================================
    // Social API
    // ========================================================================

    const social = {
        /**
         * Update location
         * @param {number} latitude - Latitude
         * @param {number} longitude - Longitude
         */
        async updateLocation(latitude, longitude) {
            return put('/social/location', { latitude, longitude });
        },

        /**
         * Get nearby pets
         * @param {number} latitude - Latitude
         * @param {number} longitude - Longitude
         * @param {number} radius - Search radius in meters
         */
        async getNearbyPets(latitude, longitude, radius = 1000) {
            return get(`/social/nearby?latitude=${latitude}&longitude=${longitude}&radius=${radius}`);
        },

        /**
         * Get friends list
         * @param {string} petId - Pet ID
         */
        async getFriends(petId) {
            return get(`/social/friends?pet_id=${petId}`);
        },

        /**
         * Send friend request
         * @param {string} fromPetId - Source pet ID
         * @param {string} toPetId - Target pet ID
         * @param {string} message - Optional message
         */
        async sendFriendRequest(fromPetId, toPetId, message = '') {
            return post(`/social/friends/request?pet_id=${fromPetId}`, {
                target_pet_id: toPetId,
                message
            });
        },

        /**
         * Get pending friend requests
         * @param {string} petId - Pet ID
         */
        async getFriendRequests(petId) {
            return get(`/social/friends/requests?pet_id=${petId}`);
        },

        /**
         * Accept friend request
         * @param {string} requestId - Request ID
         */
        async acceptFriendRequest(requestId) {
            return post(`/social/friends/requests/${requestId}/accept`);
        },

        /**
         * Decline friend request
         * @param {string} requestId - Request ID
         */
        async declineFriendRequest(requestId) {
            return post(`/social/friends/requests/${requestId}/decline`);
        },

        /**
         * Remove friend
         * @param {string} petId - Your pet ID
         * @param {string} friendPetId - Friend's pet ID
         */
        async removeFriend(petId, friendPetId) {
            return del(`/social/friends/${friendPetId}?my_pet_id=${petId}`);
        },

        /**
         * Perform social interaction
         * @param {string} petId - Your pet ID
         * @param {string} targetPetId - Target pet ID
         * @param {string} type - Interaction type (wave, play, etc.)
         */
        async interact(petId, targetPetId, type) {
            return post(`/social/interact?pet_id=${petId}`, {
                target_pet_id: targetPetId,
                type
            });
        },

        /**
         * Search for pets
         * @param {string} query - Search query
         */
        async searchPets(query) {
            return get(`/social/search?q=${encodeURIComponent(query)}`);
        },

        /**
         * Get public pet profile
         * @param {string} petId - Pet ID
         */
        async getPetProfile(petId) {
            return get(`/social/pets/${petId}`);
        }
    };

    // ========================================================================
    // Feed API
    // ========================================================================

    const feed = {
        /**
         * Get social feed
         * @param {number} page - Page number
         * @param {number} pageSize - Items per page
         */
        async get(page = 1, pageSize = 20) {
            return get(`/feed?page=${page}&page_size=${pageSize}`);
        },

        /**
         * Create a new post
         * @param {string} content - Post content
         * @param {string} imageUrl - Optional image URL
         * @param {string[]} taggedPets - Tagged pet IDs
         */
        async post(content, imageUrl = '', taggedPets = []) {
            return post('/feed', { content, image_url: imageUrl, tagged_pets: taggedPets });
        },

        /**
         * Get a specific feed item
         * @param {string} itemId - Feed item ID
         */
        async getItem(itemId) {
            return get(`/feed/${itemId}`);
        },

        /**
         * Delete a feed post
         * @param {string} itemId - Feed item ID
         */
        async deletePost(itemId) {
            return del(`/feed/${itemId}`);
        },

        /**
         * Like a feed item
         * @param {string} itemId - Feed item ID
         */
        async like(itemId) {
            return post(`/feed/${itemId}/like`);
        },

        /**
         * Unlike a feed item
         * @param {string} itemId - Feed item ID
         */
        async unlike(itemId) {
            return del(`/feed/${itemId}/like`);
        }
    };

    // ========================================================================
    // Events API
    // ========================================================================

    const events = {
        /**
         * Get active events
         * @param {string} filter - Filter type (all, nearby, joined)
         */
        async list(filter = 'all') {
            return get(`/events?filter=${filter}`);
        },

        /**
         * Get nearby events
         * @param {number} latitude - Latitude
         * @param {number} longitude - Longitude
         * @param {number} radius - Radius in meters
         */
        async getNearby(latitude, longitude, radius = 5000) {
            return get(`/events/nearby?latitude=${latitude}&longitude=${longitude}&radius=${radius}`);
        },

        /**
         * Get event details
         * @param {string} eventId - Event ID
         */
        async get(eventId) {
            return get(`/events/${eventId}`);
        },

        /**
         * Join an event
         * @param {string} eventId - Event ID
         */
        async join(eventId) {
            return post(`/events/${eventId}/join`);
        },

        /**
         * Leave an event
         * @param {string} eventId - Event ID
         */
        async leave(eventId) {
            return post(`/events/${eventId}/leave`);
        },

        /**
         * Get event participants
         * @param {string} eventId - Event ID
         */
        async getParticipants(eventId) {
            return get(`/events/${eventId}/participants`);
        }
    };

    // ========================================================================
    // Leaderboards API
    // ========================================================================

    const leaderboards = {
        /**
         * Get available leaderboards
         */
        async list() {
            return get('/leaderboards');
        },

        /**
         * Get leaderboard entries
         * @param {string} leaderboardId - Leaderboard ID
         * @param {number} limit - Max entries
         */
        async get(leaderboardId, limit = 100) {
            return get(`/leaderboards/${leaderboardId}?limit=${limit}`);
        },

        /**
         * Get user's rank
         * @param {string} leaderboardId - Leaderboard ID
         */
        async getRank(leaderboardId) {
            return get(`/leaderboards/${leaderboardId}/rank`);
        }
    };

    // ========================================================================
    // Memories API
    // ========================================================================

    const memories = {
        /**
         * Get pet memories
         * @param {string} petId - Pet ID
         * @param {number} page - Page number
         * @param {number} pageSize - Items per page
         */
        async list(petId, page = 1, pageSize = 20) {
            return get(`/pets/${petId}/memories?page=${page}&page_size=${pageSize}`);
        },

        /**
         * Get memory summary
         * @param {string} petId - Pet ID
         */
        async getSummary(petId) {
            return get(`/pets/${petId}/memories/summary`);
        },

        /**
         * Search memories
         * @param {string} petId - Pet ID
         * @param {string} query - Search query
         */
        async search(petId, query) {
            return get(`/pets/${petId}/memories/search?q=${encodeURIComponent(query)}`);
        },

        /**
         * Get positive memories
         * @param {string} petId - Pet ID
         */
        async getPositive(petId) {
            return get(`/pets/${petId}/memories/positive`);
        }
    };

    // ========================================================================
    // Achievements API
    // ========================================================================

    const achievements = {
        /**
         * Get all achievements
         */
        async list() {
            return get('/achievements');
        },

        /**
         * Get achievement details
         * @param {string} achievementId - Achievement ID
         */
        async get(achievementId) {
            return get(`/achievements/${achievementId}`);
        },

        /**
         * Get achievement categories
         */
        async getCategories() {
            return get('/achievements/categories');
        },

        /**
         * Get recent achievements
         */
        async getRecent() {
            return get('/achievements/recent');
        },

        /**
         * Share achievement
         * @param {string} achievementId - Achievement ID
         */
        async share(achievementId) {
            return post(`/achievements/${achievementId}/share`);
        }
    };

    // ========================================================================
    // Environment API
    // ========================================================================

    const environment = {
        /**
         * Get current environment state
         */
        async get() {
            return get('/environment');
        },

        /**
         * Get current weather
         */
        async getWeather() {
            return get('/environment/weather');
        },

        /**
         * Get weather forecast
         */
        async getForecast() {
            return get('/environment/forecast');
        }
    };

    // ========================================================================
    // Sync API
    // ========================================================================

    const sync = {
        /**
         * Synchronize with cloud
         * @param {Date} lastSyncTime - Last sync timestamp
         * @param {number} localVersion - Local data version
         * @param {object} petData - Local pet data
         */
        async sync(lastSyncTime, localVersion, petData = null) {
            return post('/sync', {
                last_sync_time: lastSyncTime.toISOString(),
                local_version: localVersion,
                pet_data: petData
            });
        },

        /**
         * Get sync status
         */
        async getStatus() {
            return get('/sync/status');
        },

        /**
         * Resolve sync conflict
         * @param {string} resolution - Resolution type (keep_local, keep_server, merge)
         * @param {object} mergedData - Merged data if using merge
         */
        async resolveConflict(resolution, mergedData = null) {
            return post('/sync/resolve', { resolution, merged_data: mergedData });
        },

        /**
         * Create backup
         */
        async createBackup() {
            return post('/sync/backup');
        },

        /**
         * List backups
         */
        async listBackups() {
            return get('/sync/backups');
        },

        /**
         * Restore from backup
         * @param {string} backupId - Backup ID
         */
        async restore(backupId) {
            return post(`/sync/restore/${backupId}`);
        }
    };

    // ========================================================================
    // Initialization
    // ========================================================================

    // Load stored tokens on init
    loadTokens();

    // ========================================================================
    // Public API
    // ========================================================================

    return {
        // Configuration
        config,
        setBaseURL(url) { config.baseURL = url; },

        // Core HTTP methods
        request,
        get,
        post,
        put,
        del,

        // Error class
        APIError,

        // API modules
        auth,
        pets,
        interactions,
        social,
        feed,
        events,
        leaderboards,
        memories,
        achievements,
        environment,
        sync
    };
})();

// Export for module systems
if (typeof module !== 'undefined' && module.exports) {
    module.exports = GochiAPI;
}
