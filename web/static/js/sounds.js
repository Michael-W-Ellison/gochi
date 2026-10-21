// Sounds, made in the browser with the Web Audio API: there are no sound
// files and nothing is downloaded. Sound is on at half volume to start.
// The player can turn it off, change the volume and set quiet hours on
// this device; a parent can turn it off for the household in Privacy.

const soundDefaults = {on: true, volume: 0.5, quietFrom: -1, quietTo: -1};
let audioContext = null;

function soundSettings() {
    try {
        return Object.assign({}, soundDefaults, JSON.parse(localStorage.getItem('gochiSound') || '{}'));
    } catch (e) {
        return Object.assign({}, soundDefaults);
    }
}

function saveSoundSettings(settings) {
    try { localStorage.setItem('gochiSound', JSON.stringify(settings)); } catch (e) { /* Not remembered */ }
}

// inQuietHours reports whether the hour is in the quiet hours (which can
// run past midnight)
function inQuietHours(settings, hour) {
    const from = settings.quietFrom, to = settings.quietTo;
    if (from < 0 || to < 0 || from === to) return false;
    return from < to ? hour >= from && hour < to : hour >= from || hour < to;
}

function soundAllowed() {
    const s = soundSettings();
    return s.on && s.volume > 0 && !(lastStatus && lastStatus.sound_off) && !inQuietHours(s, new Date().getHours());
}

function audio() {
    if (!audioContext) {
        const AC = window.AudioContext || window.webkitAudioContext;
        if (!AC) return null;
        audioContext = new AC();
    }
    if (audioContext.state === 'suspended') audioContext.resume();
    return audioContext;
}

// tone plays a note that can slide in pitch; times are in seconds
function tone(ctx, out, {freq, to, start = 0, dur = 0.15, type = 'sine', gain = 0.5}) {
    const t = ctx.currentTime + start;
    const osc = ctx.createOscillator();
    const env = ctx.createGain();
    osc.type = type;
    osc.frequency.setValueAtTime(freq, t);
    if (to) osc.frequency.exponentialRampToValueAtTime(to, t + dur);
    env.gain.setValueAtTime(0.0001, t);
    env.gain.exponentialRampToValueAtTime(gain, t + Math.min(0.02, dur / 3));
    env.gain.exponentialRampToValueAtTime(0.0001, t + dur);
    osc.connect(env).connect(out);
    osc.start(t);
    osc.stop(t + dur + 0.02);
}

// noise plays filtered noise, for rustles, munching and lapping
function noise(ctx, out, {start = 0, dur = 0.1, freq = 1500, gain = 0.4}) {
    const t = ctx.currentTime + start;
    const buffer = ctx.createBuffer(1, Math.ceil(ctx.sampleRate * dur), ctx.sampleRate);
    const data = buffer.getChannelData(0);
    for (let i = 0; i < data.length; i++) data[i] = Math.random() * 2 - 1;
    const src = ctx.createBufferSource();
    src.buffer = buffer;
    const filter = ctx.createBiquadFilter();
    filter.type = 'bandpass';
    filter.frequency.value = freq;
    const env = ctx.createGain();
    env.gain.setValueAtTime(gain, t);
    env.gain.exponentialRampToValueAtTime(0.0001, t + dur);
    src.connect(filter).connect(env).connect(out);
    src.start(t);
}

// The sounds, each a short recipe of notes and noise
const soundRecipes = {
    purr: (c, o) => { for (let i = 0; i < 8; i++) tone(c, o, {freq: 26, start: i * 0.09, dur: 0.08, type: 'sawtooth', gain: 0.25}); },
    meow_happy: (c, o) => tone(c, o, {freq: 600, to: 900, dur: 0.35, type: 'triangle'}),
    meow_sad: (c, o) => tone(c, o, {freq: 700, to: 420, dur: 0.5, type: 'triangle', gain: 0.4}),
    meow_ask: (c, o) => { tone(c, o, {freq: 500, to: 800, dur: 0.2, type: 'triangle'}); tone(c, o, {freq: 800, to: 650, start: 0.2, dur: 0.15, type: 'triangle'}); },
    bark_happy: (c, o) => { tone(c, o, {freq: 380, to: 260, dur: 0.12, type: 'square', gain: 0.3}); tone(c, o, {freq: 400, to: 270, start: 0.18, dur: 0.12, type: 'square', gain: 0.3}); },
    bark_alert: (c, o) => tone(c, o, {freq: 300, to: 180, dur: 0.2, type: 'square', gain: 0.35}),
    bark_sad: (c, o) => tone(c, o, {freq: 260, to: 160, dur: 0.35, type: 'square', gain: 0.2}),
    whine: (c, o) => tone(c, o, {freq: 900, to: 1300, dur: 0.6, type: 'sine', gain: 0.25}),
    munch: (c, o) => { for (let i = 0; i < 4; i++) noise(c, o, {start: i * 0.14, dur: 0.07, freq: 900}); },
    lap: (c, o) => { for (let i = 0; i < 5; i++) noise(c, o, {start: i * 0.1, dur: 0.05, freq: 2500, gain: 0.3}); },
    squeak: (c, o) => tone(c, o, {freq: 1400, to: 1900, dur: 0.12, type: 'sine', gain: 0.35}),
    bell: (c, o) => { tone(c, o, {freq: 1320, dur: 0.8, gain: 0.3}); tone(c, o, {freq: 1980, dur: 0.6, gain: 0.12}); },
    fanfare: (c, o) => [523, 659, 784, 1047].forEach((f, i) => tone(c, o, {freq: f, start: i * 0.12, dur: i === 3 ? 0.45 : 0.12, type: 'triangle', gain: 0.35})),
    chime: (c, o) => { tone(c, o, {freq: 1047, dur: 0.15, gain: 0.3}); tone(c, o, {freq: 1568, start: 0.08, dur: 0.25, gain: 0.25}); },
    page: (c, o) => noise(c, o, {dur: 0.18, freq: 3000, gain: 0.25}),
    alert: (c, o) => { tone(c, o, {freq: 660, dur: 0.18, gain: 0.25}); tone(c, o, {freq: 520, start: 0.2, dur: 0.25, gain: 0.25}); },
    throw: (c, o) => noise(c, o, {dur: 0.3, freq: 600, gain: 0.3}),
    catch: (c, o) => { tone(c, o, {freq: 880, dur: 0.1, gain: 0.3}); tone(c, o, {freq: 1175, start: 0.1, dur: 0.2, gain: 0.3}); },
    rustle: (c, o) => { for (let i = 0; i < 3; i++) noise(c, o, {start: i * 0.12, dur: 0.1, freq: 4000, gain: 0.2}); },
};

// playSound plays one of the sounds, if sound is allowed now
function playSound(name) {
    const recipe = soundRecipes[name];
    if (!recipe || !soundAllowed()) return;
    const ctx = audio();
    if (!ctx) return;
    const out = ctx.createGain();
    out.gain.value = soundSettings().volume;
    out.connect(ctx.destination);
    try { recipe(ctx, out); } catch (e) { /* A sound is never worth an error */ }
}

// A pet's voice for how it feels: happy, sad or asking
function petSound(mood) {
    const dog = lastStatus && lastStatus.species === 'dog';
    playSound(dog ? {happy: 'bark_happy', sad: 'whine', ask: 'bark_alert'}[mood] : {happy: 'purr', sad: 'meow_sad', ask: 'meow_ask'}[mood]);
}

// The sound settings, in the Settings panel
function renderSoundSettings(p) {
    const s = soundSettings();
    cloudText(p, 'Sound', 'section-title');
    if (lastStatus && lastStatus.sound_off) {
        cloudText(p, 'A parent has turned sound off in Privacy.', 'panel-text cloud-note');
        return;
    }
    const on = el('label', 'panel-text');
    const box = el('input');
    box.type = 'checkbox';
    box.checked = s.on;
    on.append(box, ' Sounds on');
    const volume = el('input');
    volume.type = 'range';
    volume.min = 0; volume.max = 100; volume.value = Math.round(s.volume * 100);
    volume.setAttribute('aria-label', 'Volume');
    const hours = el('div', 'panel-text', 'Quiet hours: ');
    const hourSelect = (value, label) => {
        const sel = el('select', 'show-level');
        sel.setAttribute('aria-label', label);
        const off = el('option', '', 'none');
        off.value = -1;
        sel.append(off);
        for (let h = 0; h < 24; h++) {
            const o = el('option', '', (h < 10 ? '0' : '') + h + ':00');
            o.value = h;
            sel.append(o);
        }
        sel.value = value;
        return sel;
    };
    const from = hourSelect(s.quietFrom, 'Quiet from'), to = hourSelect(s.quietTo, 'Quiet until');
    hours.append(from, ' to ', to);
    const save = () => {
        saveSoundSettings({on: box.checked, volume: volume.value / 100, quietFrom: Number(from.value), quietTo: Number(to.value)});
        playSound('chime');
    };
    [box, volume, from, to].forEach(c => { c.onchange = save; });
    const volumeRow = el('label', 'panel-text', 'Volume ');
    volumeRow.append(volume);
    p.append(on, volumeRow, hours);
    cloudText(p, 'Sounds are made in your browser; nothing is downloaded. These settings are kept on this device.',
        'panel-text cloud-note');
}

// Browsers start sound only after the player touches the page
document.addEventListener('pointerdown', () => { if (audioContext && audioContext.state === 'suspended') audioContext.resume(); });
