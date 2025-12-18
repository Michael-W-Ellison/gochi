/**
 * Gochi Pet Rendering Module
 * Handles pet display, animations, and visual state management
 */

const GochiPet = (function() {
    'use strict';

    // ========================================================================
    // Configuration
    // ========================================================================

    const config = {
        animationDuration: 2000,
        blinkInterval: 4000,
        idleAnimationInterval: 5000,
        effectDuration: 1000,
        moodBubbleDuration: 3000
    };

    // ========================================================================
    // State
    // ========================================================================

    let currentPet = null;
    let currentBehavior = 'idle';
    let animationFrame = null;
    let blinkTimer = null;
    let idleTimer = null;

    // DOM Elements
    let elements = {
        sprite: null,
        body: null,
        eyes: null,
        mouth: null,
        shadow: null,
        effects: null,
        moodBubble: null,
        moodText: null,
        environmentBg: null
    };

    // ========================================================================
    // Behavior States
    // ========================================================================

    const behaviors = {
        idle: {
            animation: 'petIdle',
            eyeStyle: 'normal',
            mouthStyle: 'neutral'
        },
        sleeping: {
            animation: 'petSleep',
            eyeStyle: 'closed',
            mouthStyle: 'neutral'
        },
        eating: {
            animation: 'petEat',
            eyeStyle: 'happy',
            mouthStyle: 'eating'
        },
        playing: {
            animation: 'petPlay',
            eyeStyle: 'excited',
            mouthStyle: 'happy'
        },
        happy: {
            animation: 'petIdle',
            eyeStyle: 'happy',
            mouthStyle: 'happy'
        },
        excited: {
            animation: 'petExcited',
            eyeStyle: 'excited',
            mouthStyle: 'happy'
        },
        sick: {
            animation: 'petIdle',
            eyeStyle: 'sad',
            mouthStyle: 'sad'
        },
        distressed: {
            animation: 'petDistressed',
            eyeStyle: 'sad',
            mouthStyle: 'sad'
        },
        exploring: {
            animation: 'petIdle',
            eyeStyle: 'curious',
            mouthStyle: 'neutral'
        },
        grooming: {
            animation: 'petIdle',
            eyeStyle: 'content',
            mouthStyle: 'neutral'
        }
    };

    // ========================================================================
    // Mood Icons
    // ========================================================================

    const moodIcons = {
        ecstatic: '😄',
        happy: '😊',
        content: '🙂',
        neutral: '😐',
        sad: '😢',
        distressed: '😰',
        angry: '😠',
        scared: '😨',
        excited: '🤩',
        loving: '🥰',
        sleepy: '😴',
        hungry: '😋'
    };

    // ========================================================================
    // Effect Emojis
    // ========================================================================

    const effects = {
        feed: ['🍖', '🥕', '🍎', '🦴'],
        pet: ['💕', '💖', '✨', '💫'],
        play: ['⚽', '🎾', '🎮', '🎪'],
        groom: ['✨', '🫧', '🧼', '💧'],
        train: ['💪', '⭐', '🎯', '📚'],
        medical: ['💊', '❤️‍🩹', '🩺', '💉'],
        reward: ['⭐', '🌟', '💎', '🏆'],
        levelUp: ['🎉', '🎊', '✨', '🌈'],
        happy: ['💖', '💕', '😊', '🌸'],
        sad: ['💧', '😢', '🥺'],
        sleep: ['💤', 'zzz', '😴']
    };

    // ========================================================================
    // Initialization
    // ========================================================================

    /**
     * Initialize the pet module
     */
    function init() {
        cacheElements();
        setupEventListeners();
        startIdleAnimations();
    }

    /**
     * Cache DOM elements for performance
     */
    function cacheElements() {
        elements.sprite = document.getElementById('pet-sprite');
        elements.body = document.querySelector('.pet-body');
        elements.eyes = document.querySelector('.pet-eyes');
        elements.mouth = document.querySelector('.pet-mouth');
        elements.shadow = document.querySelector('.pet-shadow');
        elements.effects = document.getElementById('pet-effects');
        elements.moodBubble = document.getElementById('mood-bubble');
        elements.moodText = document.getElementById('mood-text');
        elements.environmentBg = document.getElementById('environment-bg');
    }

    /**
     * Setup event listeners
     */
    function setupEventListeners() {
        // Click on pet for random reaction
        if (elements.sprite) {
            elements.sprite.addEventListener('click', handlePetClick);
        }
    }

    /**
     * Start idle animation timers
     */
    function startIdleAnimations() {
        // Random blink
        blinkTimer = setInterval(() => {
            if (currentBehavior !== 'sleeping') {
                blink();
            }
        }, config.blinkInterval + Math.random() * 2000);

        // Random idle animations
        idleTimer = setInterval(() => {
            if (currentBehavior === 'idle') {
                playIdleAnimation();
            }
        }, config.idleAnimationInterval);
    }

    // ========================================================================
    // Pet Display
    // ========================================================================

    /**
     * Set the current pet data
     * @param {object} pet - Pet data from API
     */
    function setPet(pet) {
        currentPet = pet;
        updateDisplay();
    }

    /**
     * Update the pet display based on current state
     */
    function updateDisplay() {
        if (!currentPet) return;

        // Update behavior/animation
        setBehavior(currentPet.current_behavior || 'idle');

        // Update pet color based on personality or customization
        updatePetColors();

        // Update environment based on time/weather
        updateEnvironment();
    }

    /**
     * Set the pet's current behavior/animation state
     * @param {string} behavior - Behavior state name
     */
    function setBehavior(behavior) {
        const normalizedBehavior = behavior.toLowerCase().replace(/_/g, '');
        const behaviorConfig = behaviors[normalizedBehavior] || behaviors.idle;

        // Remove all behavior classes
        if (elements.sprite) {
            Object.keys(behaviors).forEach(b => {
                elements.sprite.classList.remove(b);
            });

            // Add current behavior class
            elements.sprite.classList.add(normalizedBehavior);
        }

        // Update facial expressions
        setEyeStyle(behaviorConfig.eyeStyle);
        setMouthStyle(behaviorConfig.mouthStyle);

        currentBehavior = normalizedBehavior;
    }

    /**
     * Set eye style
     * @param {string} style - Eye style (normal, happy, sad, closed, excited, curious, content)
     */
    function setEyeStyle(style) {
        if (!elements.eyes) return;

        elements.eyes.className = 'pet-eyes';
        elements.eyes.classList.add(`eyes-${style}`);
    }

    /**
     * Set mouth style
     * @param {string} style - Mouth style (neutral, happy, sad, eating)
     */
    function setMouthStyle(style) {
        if (!elements.mouth) return;

        elements.mouth.className = 'pet-mouth';
        elements.mouth.classList.add(`mouth-${style}`);
    }

    /**
     * Update pet colors based on customization
     */
    function updatePetColors() {
        if (!elements.body || !currentPet) return;

        // Default colors if no customization
        const primaryColor = currentPet.appearance?.primary_color || '#6c5ce7';
        const secondaryColor = currentPet.appearance?.secondary_color || '#a29bfe';

        elements.body.style.background = primaryColor;
        elements.body.style.boxShadow = `inset -10px -10px 20px rgba(0,0,0,0.1), 0 0 20px ${secondaryColor}40`;
    }

    /**
     * Update environment background based on time and weather
     */
    function updateEnvironment() {
        if (!elements.environmentBg) return;

        const hour = new Date().getHours();
        const isNight = hour < 6 || hour >= 20;

        if (isNight) {
            elements.environmentBg.classList.add('night');
        } else {
            elements.environmentBg.classList.remove('night');
        }
    }

    // ========================================================================
    // Animations
    // ========================================================================

    /**
     * Play blink animation
     */
    function blink() {
        if (!elements.eyes) return;

        elements.eyes.classList.add('blinking');
        setTimeout(() => {
            elements.eyes.classList.remove('blinking');
        }, 200);
    }

    /**
     * Play random idle animation
     */
    function playIdleAnimation() {
        const animations = ['wiggle', 'look-around', 'yawn'];
        const randomAnim = animations[Math.floor(Math.random() * animations.length)];

        if (elements.sprite) {
            elements.sprite.classList.add(randomAnim);
            setTimeout(() => {
                elements.sprite.classList.remove(randomAnim);
            }, 1000);
        }
    }

    /**
     * Handle click on pet
     */
    function handlePetClick() {
        // Play a happy reaction
        playEffect('pet');
        showMoodBubble('💖');

        // Trigger a quick animation
        if (elements.sprite) {
            elements.sprite.classList.add('clicked');
            setTimeout(() => {
                elements.sprite.classList.remove('clicked');
            }, 300);
        }
    }

    // ========================================================================
    // Effects
    // ========================================================================

    /**
     * Play an interaction effect
     * @param {string} type - Effect type (feed, pet, play, etc.)
     */
    function playEffect(type) {
        if (!elements.effects) return;

        const effectEmojis = effects[type] || effects.happy;
        const count = Math.floor(Math.random() * 3) + 2;

        for (let i = 0; i < count; i++) {
            setTimeout(() => {
                createEffectParticle(effectEmojis);
            }, i * 150);
        }
    }

    /**
     * Create a single effect particle
     * @param {string[]} emojis - Array of emoji options
     */
    function createEffectParticle(emojis) {
        if (!elements.effects) return;

        const particle = document.createElement('div');
        particle.className = 'effect';
        particle.textContent = emojis[Math.floor(Math.random() * emojis.length)];

        // Random position around the pet
        const angle = Math.random() * Math.PI * 2;
        const distance = 50 + Math.random() * 50;
        const x = Math.cos(angle) * distance;
        const y = Math.sin(angle) * distance;

        particle.style.left = `calc(50% + ${x}px)`;
        particle.style.top = `calc(50% + ${y}px)`;
        particle.style.fontSize = `${1.5 + Math.random()}rem`;

        elements.effects.appendChild(particle);

        // Remove after animation
        setTimeout(() => {
            particle.remove();
        }, config.effectDuration);
    }

    /**
     * Show mood bubble with message
     * @param {string} message - Message to display
     */
    function showMoodBubble(message) {
        if (!elements.moodBubble || !elements.moodText) return;

        elements.moodText.textContent = message;
        elements.moodBubble.classList.add('visible');

        setTimeout(() => {
            elements.moodBubble.classList.remove('visible');
        }, config.moodBubbleDuration);
    }

    // ========================================================================
    // Vitals Display
    // ========================================================================

    /**
     * Update vitals display
     * @param {object} vitals - Vitals data from API
     */
    function updateVitals(vitals) {
        const vitalsMap = {
            health: vitals.health,
            energy: vitals.energy,
            happiness: vitals.happiness,
            nutrition: vitals.nutrition,
            cleanliness: vitals.cleanliness
        };

        Object.entries(vitalsMap).forEach(([key, value]) => {
            const bar = document.getElementById(`${key}-bar`);
            const valueEl = document.getElementById(`${key}-value`);

            if (bar) {
                bar.style.width = `${value * 100}%`;

                // Add warning class if low
                if (value < 0.3) {
                    bar.classList.add('critical');
                } else if (value < 0.5) {
                    bar.classList.add('warning');
                } else {
                    bar.classList.remove('critical', 'warning');
                }
            }

            if (valueEl) {
                valueEl.textContent = `${Math.round(value * 100)}%`;
            }
        });
    }

    /**
     * Update emotions display
     * @param {object} emotions - Emotions data from API
     */
    function updateEmotions(emotions) {
        // Update mood icon
        const moodIcon = document.getElementById('mood-icon');
        if (moodIcon) {
            const icon = getMoodIcon(emotions.mood_description);
            moodIcon.textContent = icon;
        }

        // Update mood label
        const moodLabel = document.getElementById('mood-label');
        if (moodLabel) {
            moodLabel.textContent = emotions.mood_description || 'Content';
        }

        // Update mood trend
        const moodTrend = document.getElementById('mood-trend');
        if (moodTrend) {
            const trend = emotions.mood_trend || '';
            let trendText = '';
            if (trend.includes('improving') || trend.includes('up')) {
                trendText = '↑ Improving';
            } else if (trend.includes('declining') || trend.includes('down')) {
                trendText = '↓ Declining';
            } else {
                trendText = '→ Stable';
            }
            moodTrend.textContent = trendText;
        }

        // Update emotion chips
        const chipsContainer = document.getElementById('emotion-chips');
        if (chipsContainer) {
            chipsContainer.innerHTML = '';

            const emotionValues = [
                { name: 'Joy', value: emotions.joy },
                { name: 'Excitement', value: emotions.excitement },
                { name: 'Contentment', value: emotions.contentment },
                { name: 'Affection', value: emotions.affection },
                { name: 'Sadness', value: emotions.sadness },
                { name: 'Fear', value: emotions.fear }
            ];

            // Show top emotions
            emotionValues
                .filter(e => e.value > 0.3)
                .sort((a, b) => b.value - a.value)
                .slice(0, 3)
                .forEach(emotion => {
                    const chip = document.createElement('span');
                    chip.className = 'emotion-chip';
                    chip.textContent = `${emotion.name} ${Math.round(emotion.value * 100)}%`;
                    chipsContainer.appendChild(chip);
                });
        }
    }

    /**
     * Get mood icon based on description
     * @param {string} description - Mood description
     * @returns {string} Emoji icon
     */
    function getMoodIcon(description) {
        const desc = (description || '').toLowerCase();

        if (desc.includes('ecstatic') || desc.includes('overjoyed')) return moodIcons.ecstatic;
        if (desc.includes('happy') || desc.includes('cheerful')) return moodIcons.happy;
        if (desc.includes('content') || desc.includes('peaceful')) return moodIcons.content;
        if (desc.includes('sad') || desc.includes('melancholy')) return moodIcons.sad;
        if (desc.includes('distress') || desc.includes('anxious')) return moodIcons.distressed;
        if (desc.includes('angry') || desc.includes('frustrated')) return moodIcons.angry;
        if (desc.includes('scared') || desc.includes('fearful')) return moodIcons.scared;
        if (desc.includes('excited') || desc.includes('energetic')) return moodIcons.excited;
        if (desc.includes('loving') || desc.includes('affection')) return moodIcons.loving;
        if (desc.includes('sleepy') || desc.includes('tired')) return moodIcons.sleepy;

        return moodIcons.neutral;
    }

    // ========================================================================
    // Interaction Animations
    // ========================================================================

    /**
     * Play feeding animation
     */
    function playFeedAnimation() {
        setBehavior('eating');
        playEffect('feed');
        showMoodBubble('😋 Yum!');

        setTimeout(() => {
            setBehavior('happy');
            setTimeout(() => {
                setBehavior('idle');
            }, 2000);
        }, 3000);
    }

    /**
     * Play petting animation
     */
    function playPetAnimation() {
        setBehavior('happy');
        playEffect('pet');
        showMoodBubble('💕');

        setTimeout(() => {
            setBehavior('idle');
        }, 3000);
    }

    /**
     * Play playing animation
     */
    function playPlayAnimation() {
        setBehavior('playing');
        playEffect('play');
        showMoodBubble('🎮 Fun!');

        setTimeout(() => {
            setBehavior('excited');
            setTimeout(() => {
                setBehavior('idle');
            }, 2000);
        }, 4000);
    }

    /**
     * Play grooming animation
     */
    function playGroomAnimation() {
        setBehavior('grooming');
        playEffect('groom');
        showMoodBubble('✨ Clean!');

        setTimeout(() => {
            setBehavior('idle');
        }, 3000);
    }

    /**
     * Play training animation
     */
    function playTrainAnimation() {
        setBehavior('excited');
        playEffect('train');
        showMoodBubble('📚 Learning!');

        setTimeout(() => {
            playEffect('levelUp');
            setBehavior('happy');
            setTimeout(() => {
                setBehavior('idle');
            }, 2000);
        }, 3000);
    }

    /**
     * Play medical animation
     */
    function playMedicalAnimation() {
        setBehavior('distressed');
        playEffect('medical');

        setTimeout(() => {
            showMoodBubble('❤️‍🩹 Better!');
            setBehavior('idle');
        }, 2000);
    }

    /**
     * Play animation for interaction type
     * @param {string} type - Interaction type
     */
    function playInteractionAnimation(type) {
        switch (type) {
            case 'feed':
            case 'feeding':
                playFeedAnimation();
                break;
            case 'pet':
            case 'petting':
                playPetAnimation();
                break;
            case 'play':
            case 'playing':
                playPlayAnimation();
                break;
            case 'groom':
            case 'grooming':
                playGroomAnimation();
                break;
            case 'train':
            case 'training':
                playTrainAnimation();
                break;
            case 'medical':
            case 'medical_care':
                playMedicalAnimation();
                break;
            default:
                playEffect('happy');
        }
    }

    // ========================================================================
    // Stats Display
    // ========================================================================

    /**
     * Update quick stats
     * @param {object} pet - Pet data
     */
    function updateStats(pet) {
        const stats = {
            'stat-age': `${Math.floor(pet.age_days || 0)} days`,
            'stat-interactions': pet.statistics?.total_interactions || 0,
            'stat-friends': pet.statistics?.friend_count || 0,
            'stat-wellbeing': `${Math.round((pet.vitals?.wellbeing || 0) * 100)}%`
        };

        Object.entries(stats).forEach(([id, value]) => {
            const el = document.getElementById(id);
            if (el) {
                el.textContent = value;
            }
        });
    }

    /**
     * Update pet name and age display
     * @param {object} pet - Pet data
     */
    function updatePetInfo(pet) {
        const nameEl = document.getElementById('pet-name');
        const ageEl = document.getElementById('pet-age');
        const behaviorEl = document.getElementById('behavior-indicator');

        if (nameEl) {
            nameEl.textContent = pet.name || 'Unknown';
        }

        if (ageEl) {
            ageEl.textContent = `${Math.floor(pet.age_days || 0)} days old`;
        }

        if (behaviorEl) {
            const behavior = pet.current_behavior || 'Idle';
            behaviorEl.textContent = behavior.replace(/_/g, ' ');
        }
    }

    // ========================================================================
    // Cleanup
    // ========================================================================

    /**
     * Cleanup timers and animations
     */
    function destroy() {
        if (blinkTimer) {
            clearInterval(blinkTimer);
        }
        if (idleTimer) {
            clearInterval(idleTimer);
        }
        if (animationFrame) {
            cancelAnimationFrame(animationFrame);
        }
    }

    // ========================================================================
    // Public API
    // ========================================================================

    return {
        init,
        destroy,
        setPet,
        updateDisplay,
        setBehavior,
        updateVitals,
        updateEmotions,
        updateStats,
        updatePetInfo,
        playEffect,
        showMoodBubble,
        playInteractionAnimation,
        getMoodIcon,
        moodIcons,
        effects
    };
})();

// Export for module systems
if (typeof module !== 'undefined' && module.exports) {
    module.exports = GochiPet;
}
