/**
 * Gochi WebSocket Handler
 * Manages real-time communication with the server
 */

const GochiWebSocket = (function() {
    'use strict';

    // ========================================================================
    // Configuration
    // ========================================================================

    const config = {
        reconnectInterval: 3000,
        maxReconnectAttempts: 10,
        heartbeatInterval: 30000,
        connectionTimeout: 10000
    };

    // ========================================================================
    // State
    // ========================================================================

    let socket = null;
    let isConnected = false;
    let reconnectAttempts = 0;
    let heartbeatTimer = null;
    let reconnectTimer = null;
    let connectionPromise = null;

    // Event handlers
    const eventHandlers = {
        'pet_update': [],
        'vitals_update': [],
        'emotion_update': [],
        'nearby_pet': [],
        'friend_request': [],
        'social_interaction': [],
        'feed_item': [],
        'event_update': [],
        'achievement': [],
        'notification': [],
        'connected': [],
        'disconnected': [],
        'error': []
    };

    // ========================================================================
    // Connection Management
    // ========================================================================

    /**
     * Connect to WebSocket server
     * @param {string} petId - Optional pet ID for pet-specific connection
     * @returns {Promise<void>}
     */
    function connect(petId = null) {
        if (connectionPromise) {
            return connectionPromise;
        }

        connectionPromise = new Promise((resolve, reject) => {
            const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            const host = window.location.host;
            const path = petId ? `/api/v1/ws/pet/${petId}` : '/api/v1/ws';
            const url = `${protocol}//${host}${path}`;

            // Add auth token if available
            const token = localStorage.getItem('gochi_auth_token');
            const wsUrl = token ? `${url}?token=${token}` : url;

            console.log('[WebSocket] Connecting to:', url);

            try {
                socket = new WebSocket(wsUrl);
            } catch (error) {
                connectionPromise = null;
                reject(error);
                return;
            }

            // Connection timeout
            const timeoutId = setTimeout(() => {
                if (!isConnected) {
                    socket.close();
                    connectionPromise = null;
                    reject(new Error('Connection timeout'));
                }
            }, config.connectionTimeout);

            socket.onopen = () => {
                clearTimeout(timeoutId);
                isConnected = true;
                reconnectAttempts = 0;
                connectionPromise = null;

                console.log('[WebSocket] Connected');
                startHeartbeat();
                emit('connected', { timestamp: Date.now() });
                resolve();
            };

            socket.onclose = (event) => {
                clearTimeout(timeoutId);
                isConnected = false;
                connectionPromise = null;
                stopHeartbeat();

                console.log('[WebSocket] Disconnected:', event.code, event.reason);
                emit('disconnected', { code: event.code, reason: event.reason });

                // Auto-reconnect if not intentionally closed
                if (event.code !== 1000 && reconnectAttempts < config.maxReconnectAttempts) {
                    scheduleReconnect(petId);
                }
            };

            socket.onerror = (error) => {
                console.error('[WebSocket] Error:', error);
                emit('error', { error });
            };

            socket.onmessage = (event) => {
                handleMessage(event.data);
            };
        });

        return connectionPromise;
    }

    /**
     * Disconnect from WebSocket server
     */
    function disconnect() {
        if (reconnectTimer) {
            clearTimeout(reconnectTimer);
            reconnectTimer = null;
        }

        stopHeartbeat();

        if (socket) {
            socket.close(1000, 'Client disconnect');
            socket = null;
        }

        isConnected = false;
        reconnectAttempts = 0;
    }

    /**
     * Schedule reconnection attempt
     * @param {string} petId - Optional pet ID
     */
    function scheduleReconnect(petId) {
        if (reconnectTimer) {
            return;
        }

        reconnectAttempts++;
        const delay = config.reconnectInterval * Math.pow(1.5, reconnectAttempts - 1);

        console.log(`[WebSocket] Reconnecting in ${delay}ms (attempt ${reconnectAttempts}/${config.maxReconnectAttempts})`);

        reconnectTimer = setTimeout(() => {
            reconnectTimer = null;
            connect(petId).catch(error => {
                console.error('[WebSocket] Reconnection failed:', error);
            });
        }, delay);
    }

    // ========================================================================
    // Heartbeat
    // ========================================================================

    /**
     * Start heartbeat to keep connection alive
     */
    function startHeartbeat() {
        stopHeartbeat();

        heartbeatTimer = setInterval(() => {
            if (isConnected && socket && socket.readyState === WebSocket.OPEN) {
                send({ type: 'ping', timestamp: Date.now() });
            }
        }, config.heartbeatInterval);
    }

    /**
     * Stop heartbeat
     */
    function stopHeartbeat() {
        if (heartbeatTimer) {
            clearInterval(heartbeatTimer);
            heartbeatTimer = null;
        }
    }

    // ========================================================================
    // Message Handling
    // ========================================================================

    /**
     * Handle incoming message
     * @param {string} data - Raw message data
     */
    function handleMessage(data) {
        try {
            const message = JSON.parse(data);

            console.log('[WebSocket] Received:', message.type);

            // Handle pong response
            if (message.type === 'pong') {
                return;
            }

            // Emit event to handlers
            emit(message.type, message.payload);
        } catch (error) {
            console.error('[WebSocket] Failed to parse message:', error);
        }
    }

    /**
     * Send message to server
     * @param {object} message - Message object
     * @returns {boolean} Success status
     */
    function send(message) {
        if (!isConnected || !socket || socket.readyState !== WebSocket.OPEN) {
            console.warn('[WebSocket] Cannot send - not connected');
            return false;
        }

        try {
            socket.send(JSON.stringify(message));
            return true;
        } catch (error) {
            console.error('[WebSocket] Send error:', error);
            return false;
        }
    }

    // ========================================================================
    // Event System
    // ========================================================================

    /**
     * Subscribe to an event
     * @param {string} event - Event type
     * @param {function} handler - Event handler
     * @returns {function} Unsubscribe function
     */
    function on(event, handler) {
        if (!eventHandlers[event]) {
            eventHandlers[event] = [];
        }

        eventHandlers[event].push(handler);

        // Return unsubscribe function
        return () => off(event, handler);
    }

    /**
     * Unsubscribe from an event
     * @param {string} event - Event type
     * @param {function} handler - Event handler
     */
    function off(event, handler) {
        if (!eventHandlers[event]) {
            return;
        }

        const index = eventHandlers[event].indexOf(handler);
        if (index > -1) {
            eventHandlers[event].splice(index, 1);
        }
    }

    /**
     * Emit an event to all handlers
     * @param {string} event - Event type
     * @param {object} data - Event data
     */
    function emit(event, data) {
        const handlers = eventHandlers[event] || [];

        handlers.forEach(handler => {
            try {
                handler(data);
            } catch (error) {
                console.error(`[WebSocket] Handler error for ${event}:`, error);
            }
        });
    }

    /**
     * Subscribe to an event once
     * @param {string} event - Event type
     * @param {function} handler - Event handler
     */
    function once(event, handler) {
        const wrapper = (data) => {
            off(event, wrapper);
            handler(data);
        };

        on(event, wrapper);
    }

    // ========================================================================
    // Client Messages
    // ========================================================================

    /**
     * Subscribe to pet updates
     * @param {string} petId - Pet ID to subscribe to
     */
    function subscribeToPet(petId) {
        return send({
            type: 'subscribe_pet',
            payload: { pet_id: petId }
        });
    }

    /**
     * Unsubscribe from pet updates
     * @param {string} petId - Pet ID to unsubscribe from
     */
    function unsubscribeFromPet(petId) {
        return send({
            type: 'unsubscribe_pet',
            payload: { pet_id: petId }
        });
    }

    /**
     * Subscribe to nearby pets in location
     * @param {number} latitude - Latitude
     * @param {number} longitude - Longitude
     * @param {number} radius - Radius in meters
     */
    function subscribeToNearby(latitude, longitude, radius = 1000) {
        return send({
            type: 'subscribe_nearby',
            payload: { latitude, longitude, radius }
        });
    }

    /**
     * Update location for nearby features
     * @param {number} latitude - Latitude
     * @param {number} longitude - Longitude
     */
    function updateLocation(latitude, longitude) {
        return send({
            type: 'update_location',
            payload: { latitude, longitude }
        });
    }

    /**
     * Subscribe to social feed
     */
    function subscribeToFeed() {
        return send({
            type: 'subscribe_feed',
            payload: {}
        });
    }

    /**
     * Subscribe to events
     */
    function subscribeToEvents() {
        return send({
            type: 'subscribe_events',
            payload: {}
        });
    }

    // ========================================================================
    // State Getters
    // ========================================================================

    /**
     * Check if connected
     * @returns {boolean}
     */
    function getIsConnected() {
        return isConnected;
    }

    /**
     * Get connection state
     * @returns {string} - 'connecting', 'connected', 'disconnected'
     */
    function getState() {
        if (connectionPromise) {
            return 'connecting';
        }
        return isConnected ? 'connected' : 'disconnected';
    }

    /**
     * Get reconnect attempts count
     * @returns {number}
     */
    function getReconnectAttempts() {
        return reconnectAttempts;
    }

    // ========================================================================
    // Public API
    // ========================================================================

    return {
        // Configuration
        config,

        // Connection
        connect,
        disconnect,
        getIsConnected,
        getState,
        getReconnectAttempts,

        // Messaging
        send,

        // Events
        on,
        off,
        once,

        // Client actions
        subscribeToPet,
        unsubscribeFromPet,
        subscribeToNearby,
        updateLocation,
        subscribeToFeed,
        subscribeToEvents
    };
})();

// Export for module systems
if (typeof module !== 'undefined' && module.exports) {
    module.exports = GochiWebSocket;
}
