/**
 * Gochi pet drawings
 *
 * Draws a cat or dog as SVG from its look: its breed's body (build, ears,
 * muzzle, coat, legs, tail, size), its coat colors and pattern, and its
 * eye colors, in one of the pet's states (idle, happy, eating, ...), with
 * what it wears. Every value is looked up in fixed lists, so nothing from
 * a save is put into the markup as is.
 */
const GochiArt = (function () {
    'use strict';

    // Coat and eye colors (the same as the game's list in breeds.go)
    const PALETTE = {
        black: '#2b2b2e', blue: '#8592a0', red: '#d9822b', cream: '#f0dcae', chocolate: '#6b4226',
        lilac: '#b9a9ab', white: '#f7f5f0', brown: '#8a5a34', silver: '#c3c6cc', seal: '#4a3426',
        fawn: '#d2aa7c', cinnamon: '#a0602f', gold: '#e0a94a', yellow: '#ecc985', tan: '#c2884a',
        grey: '#8d8d8d', sable: '#7b5532', apricot: '#efb98a', ruddy: '#b4693a', rust: '#9c4a22',
        eye_blue: '#5aa0e6', eye_green: '#6fae53', eye_gold: '#d9a521', eye_copper: '#c56a1c',
        eye_brown: '#6b3f1f', eye_amber: '#c98a2b', eye_hazel: '#97803c',
    };
    const SKIN = '#e9b9a8'; // Hairless skin

    // Typical looks when a pet has none yet
    const DEFAULTS = {
        cat: { build: 'medium', ears: 'pointed', muzzle: 'medium', coat: 'short', legs: 'medium', tail: 'long',
            scale: 1, color: 'brown', color2: '', pattern: 'tabby', eye: 'eye_green', eye2: 'eye_green' },
        dog: { build: 'medium', ears: 'button', muzzle: 'medium', coat: 'short', legs: 'medium', tail: 'long',
            scale: 1, color: 'tan', color2: 'white', pattern: 'bicolor', eye: 'eye_brown', eye2: 'eye_brown' },
    };

    let uid = 0;

    function color(id, fallback) {
        return Object.prototype.hasOwnProperty.call(PALETTE, id) ? PALETTE[id] : fallback;
    }

    // Lighten (amount > 0) or darken (< 0) a #rrggbb color
    function shade(hex, amount) {
        const n = parseInt(hex.slice(1), 16);
        const ch = [(n >> 16) & 255, (n >> 8) & 255, n & 255].map(c =>
            Math.round(amount > 0 ? c + (255 - c) * amount : c * (1 + amount)));
        return '#' + ch.map(c => c.toString(16).padStart(2, '0')).join('');
    }

    function esc(text) {
        return String(text).replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
    }

    function oneOf(value, options, fallback) {
        return options.includes(value) ? value : fallback;
    }

    /**
     * Works out the shapes of a pet from its look
     */
    function geometry(species, look) {
        const dog = species === 'dog';
        const build = oneOf(look.build, ['slim', 'medium', 'stocky', 'large', 'lean'], 'medium');
        const legs = oneOf(look.legs, ['short', 'medium', 'long'], 'medium');
        const muzzle = oneOf(look.muzzle, ['flat', 'short', 'medium', 'long'], 'medium');
        const bodyW = { slim: 30, medium: 35, stocky: 41, large: 39, lean: 28 }[build];
        const bodyRy = { short: 32, medium: 37, long: 41 }[legs];
        const bodyCy = { short: 150, medium: 144, long: 137 }[legs];
        const legW = { slim: 12, medium: 14, stocky: 17, large: 16, lean: 10 }[build];
        const headR = muzzle === 'flat' ? 35 : dog ? 32 : 33;
        const headRy = !dog && muzzle === 'long' ? headR + 4 : headR;
        const headRx = !dog && muzzle === 'long' ? headR - 4 : headR;
        const headY = bodyCy - bodyRy - headRy + 16;
        return { dog, build, legs, muzzle, bodyW, bodyRy, bodyCy, legW, headR, headRx, headRy, headY,
            ground: 188, cx: 100 };
    }

    // The outline of the body, legs and head, used for clipping patterns
    function silhouette(g) {
        const { cx, bodyCy, bodyW, bodyRy, legW, headRx, headRy, headY, ground } = g;
        const legX = bodyW * 0.45;
        return [
            `<ellipse cx="${cx}" cy="${bodyCy}" rx="${bodyW}" ry="${bodyRy}"/>`,
            `<rect x="${cx - legX - legW / 2}" y="${bodyCy}" width="${legW}" height="${ground - bodyCy - 2}" rx="${legW / 2}"/>`,
            `<rect x="${cx + legX - legW / 2}" y="${bodyCy}" width="${legW}" height="${ground - bodyCy - 2}" rx="${legW / 2}"/>`,
            `<ellipse cx="${cx}" cy="${headY}" rx="${headRx}" ry="${headRy}"/>`,
        ].join('');
    }

    // Ears, behind (back = true) or in front of the head
    function ears(g, look, fill, inner, back) {
        const { cx, headY, headRx, headRy, dog } = g;
        const kind = oneOf(look.ears, ['pointed', 'large', 'tufted', 'rounded', 'erect', 'floppy', 'long', 'bat', 'button', 'rose'],
            dog ? 'button' : 'pointed');
        const top = headY - headRy;
        const out = [];
        for (const side of [-1, 1]) {
            const x = cx + side * headRx * 0.6;
            const tri = (w, h, lean) => {
                const bx1 = x - side * w * 0.2 - w / 2, bx2 = x - side * w * 0.2 + w / 2;
                const tipX = x + side * lean, tipY = top - h;
                return { path: `M${bx1},${top + 10} L${tipX},${tipY} L${bx2},${top + 12} Z`,
                    innerPath: `M${bx1 + w * 0.22},${top + 8} L${tipX + side * -1},${tipY + h * 0.3} L${bx2 - w * 0.22},${top + 10} Z`,
                    tipX, tipY };
            };
            if (back && ['pointed', 'large', 'tufted', 'rounded', 'erect', 'bat'].includes(kind)) {
                const t = { pointed: tri(26, 24, 6), large: tri(32, 32, 10), tufted: tri(28, 30, 6),
                    rounded: tri(24, 15, 10), erect: tri(24, 32, 4), bat: tri(34, 34, 10) }[kind];
                out.push(`<path d="${t.path}" fill="${fill}"/>`);
                out.push(`<path d="${t.innerPath}" fill="${inner}"/>`);
                if (kind === 'tufted') {
                    out.push(`<path d="M${t.tipX},${t.tipY} l${side * 2},-9 M${t.tipX},${t.tipY} l${-side * 2},-8" stroke="${fill}" stroke-width="2.5" stroke-linecap="round"/>`);
                }
            } else if (back && kind === 'rose') {
                out.push(`<path d="M${x - 10},${top + 10} q${side * 14},-16 ${20},-4 q${-side * 4},10 ${-14},10 Z" fill="${fill}"/>`);
            } else if (!back && kind === 'button') {
                out.push(`<path d="M${x - 13},${top + 8} L${x + 13},${top + 8} L${x + side * 6},${top + 26} Z" fill="${shade(fill, -0.12)}"/>`);
            } else if (!back && (kind === 'floppy' || kind === 'long')) {
                const len = kind === 'long' ? 50 : 34;
                const ex = cx + side * (headRx - 2);
                out.push(`<path d="M${ex - side * 8},${top + 12} q${side * 22},4 ${side * 16},${len} q${-side * 8},8 ${-side * 18},-4 q${-side * 4},${-len / 2} ${side * 2},${-len + 2} Z" fill="${shade(fill, -0.15)}"/>`);
            }
        }
        return out.join('');
    }

    // Coat texture at the outline: fluffy flanks and curls behind the body
    // (front = false), and a ruff under the chin in front of it
    function coatTexture(g, look, fill, front) {
        const coat = oneOf(look.coat, ['hairless', 'short', 'curly', 'wavy', 'semi', 'long', 'shaggy'], 'short');
        const { cx, bodyCy, bodyW, bodyRy, headY, headRy } = g;
        const out = [];
        if (front && (coat === 'long' || coat === 'shaggy' || coat === 'semi')) {
            const n = coat === 'semi' ? 5 : 7;
            for (let i = 0; i < n; i++) {
                const x = cx - 24 + (48 / (n - 1)) * i;
                out.push(`<circle cx="${x}" cy="${headY + headRy - 2 + (i % 2) * 4}" r="${coat === 'semi' ? 7 : 9}" fill="${fill}"/>`);
            }
        } else if (!front && (coat === 'long' || coat === 'shaggy')) {
            for (const side of [-1, 1]) {
                for (let i = 0; i < 4; i++) {
                    out.push(`<circle cx="${cx + side * (bodyW - 1)}" cy="${bodyCy - bodyRy * 0.5 + i * bodyRy * 0.35}" r="9" fill="${fill}" stroke="${shade(fill, -0.25)}" stroke-width="1"/>`);
                }
            }
        } else if (!front && coat === 'curly') {
            for (const side of [-1, 1]) {
                for (let i = 0; i < 6; i++) {
                    out.push(`<circle cx="${cx + side * (bodyW - 1)}" cy="${bodyCy - bodyRy * 0.7 + i * bodyRy * 0.28}" r="6.5" fill="${fill}" stroke="${shade(fill, -0.25)}" stroke-width="1"/>`);
                }
            }
            for (let i = 0; i < 5; i++) {
                out.push(`<circle cx="${cx - 16 + i * 8}" cy="${g.headY - g.headRy + 2}" r="7" fill="${fill}" stroke="${shade(fill, -0.25)}" stroke-width="1"/>`);
            }
        }
        return out.join('');
    }

    // Lines drawn over the coat: waves and wrinkles
    function coatLines(g, look) {
        const coat = look.coat;
        const { cx, bodyCy, headY, headRy } = g;
        if (coat === 'wavy') {
            return `<path d="M${cx - 20},${bodyCy - 10} q5,-4 10,0 q5,4 10,0 q5,-4 10,0 q5,4 10,0 M${cx - 20},${bodyCy + 6} q5,-4 10,0 q5,4 10,0 q5,-4 10,0 q5,4 10,0" stroke="rgba(0,0,0,0.18)" fill="none" stroke-width="1.5"/>`;
        }
        if (coat === 'hairless') {
            return `<path d="M${cx - 9},${headY - headRy + 12} q9,-4 18,0 M${cx - 7},${headY - headRy + 17} q7,-3 14,0 M${cx - 18},${bodyCy - 14} q8,4 16,0" stroke="rgba(120,60,50,0.35)" fill="none" stroke-width="1.5"/>`;
        }
        return '';
    }

    // The coat pattern, clipped to the silhouette
    function pattern(g, look, base, second, white) {
        const { cx, bodyCy, bodyW, bodyRy, headY, headRx, headRy, ground, legW } = g;
        const p = look.pattern;
        const dark = shade(base, -0.35);
        const legX = bodyW * 0.45;
        const paws = fill => [-1, 1].map(side =>
            `<rect x="${cx + side * legX - legW / 2 - 1}" y="${ground - 16}" width="${legW + 2}" height="16" fill="${fill}"/>`).join('');
        const chest = fill => `<ellipse cx="${cx}" cy="${bodyCy + 4}" rx="${bodyW * 0.45}" ry="${bodyRy * 0.8}" fill="${fill}"/>`;
        const muzzle = fill => `<ellipse cx="${cx}" cy="${headY + headRy * 0.45}" rx="${headRx * 0.55}" ry="${headRy * 0.45}" fill="${fill}"/>`;
        const blaze = fill => `<path d="M${cx - 5},${headY - headRy} L${cx + 5},${headY - headRy} L${cx + 8},${headY + 6} L${cx - 8},${headY + 6} Z" fill="${fill}"/>`;
        const spots = (fill, n, r, seed) => {
            let out = '';
            for (let i = 0; i < n; i++) {
                const a = (i * 137.5 + seed) * Math.PI / 180;
                const d = 0.25 + ((i * 53) % 70) / 100;
                const x = cx + Math.cos(a) * bodyW * d;
                const y = bodyCy + Math.sin(a) * bodyRy * d;
                out += `<ellipse cx="${x.toFixed(1)}" cy="${y.toFixed(1)}" rx="${r}" ry="${(r * 0.75).toFixed(1)}" fill="${fill}"/>`;
            }
            return out;
        };
        const stripes = (fill, n, w) => {
            let out = '';
            for (let i = 0; i < n; i++) {
                const y = bodyCy - bodyRy + 10 + i * (bodyRy * 2 - 10) / n;
                out += `<path d="M${cx - bodyW},${y} q${bodyW * 0.5},${-6} ${bodyW * 0.6},2 M${cx + bodyW},${y} q${-bodyW * 0.5},${-6} ${-bodyW * 0.6},2" stroke="${fill}" stroke-width="${w}" fill="none" stroke-linecap="round"/>`;
            }
            out += `<path d="M${cx - 10},${headY - headRy + 10} l5,8 l5,-8 l5,8 l5,-8" stroke="${fill}" stroke-width="${w * 0.8}" fill="none" stroke-linecap="round"/>`;
            return out;
        };
        switch (p) {
        case 'bicolor': return chest(white) + muzzle(white) + paws(white);
        case 'tuxedo': return chest(white) + paws(white) + `<ellipse cx="${cx}" cy="${headY + headRy * 0.6}" rx="${headRx * 0.35}" ry="${headRy * 0.3}" fill="${white}"/>`;
        case 'tabby': return stripes(dark, 5, 4);
        case 'ticked': return spots(shade(base, -0.2), 26, 1.6, 7) + stripes(shade(base, -0.15), 2, 2);
        case 'spotted': return spots(shade(second || base, -0.45), 16, 5, 11) + spots(shade(base, -0.4), 6, 3, 40);
        case 'point':
        case 'mitted': {
            const pt = second || dark;
            const head = `<ellipse cx="${cx}" cy="${headY + headRy * 0.35}" rx="${headRx * 0.62}" ry="${headRy * 0.55}" fill="${pt}" opacity="0.85"/>`;
            const legsDark = [-1, 1].map(side => `<rect x="${cx + side * legX - legW / 2 - 1}" y="${bodyCy + 6}" width="${legW + 2}" height="${ground - bodyCy}" fill="${pt}" opacity="0.8"/>`).join('');
            return head + legsDark + (p === 'mitted' ? paws(white) + `<ellipse cx="${cx}" cy="${bodyCy + 14}" rx="${bodyW * 0.3}" ry="${bodyRy * 0.5}" fill="${white}"/>` : '');
        }
        case 'tortie': return spots(second, 9, 9, 3) + `<path d="M${cx},${headY - headRy} L${cx + headRx},${headY} L${cx},${headY + headRy} Z" fill="${second}"/>`;
        case 'calico': return chest(white) + muzzle(white) + paws(white) + spots(second, 5, 9, 5) +
            `<ellipse cx="${cx - headRx * 0.5}" cy="${headY - headRy * 0.3}" rx="${headRx * 0.45}" ry="${headRy * 0.45}" fill="${second}"/>`;
        case 'van': return ''; // Drawn as a white coat with color on the head and tail (see draw)
        case 'merle': return spots(dark, 12, 7, 17) + spots(shade(base, -0.5), 6, 4, 61) + chest(white) + paws(white);
        case 'double_merle': return spots(base, 6, 6, 23);
        case 'brindle': {
            let out = '';
            for (let i = 0; i < 9; i++) {
                const x = cx - bodyW + i * bodyW * 0.25;
                out += `<path d="M${x},${bodyCy - bodyRy} q4,${bodyRy} -2,${bodyRy * 2}" stroke="${dark}" stroke-width="2.5" fill="none" opacity="0.7"/>`;
            }
            return out + (look.pattern === 'brindle' && g.dog ? '' : '');
        }
        case 'saddle': // Tan with a black saddle and muzzle
            return `<rect x="0" y="0" width="200" height="200" fill="${second}"/>` +
                `<path d="M${cx - bodyW - 2},${bodyCy + 4} Q${cx - bodyW},${bodyCy - bodyRy - 4} ${cx},${bodyCy - bodyRy - 2} Q${cx + bodyW},${bodyCy - bodyRy - 4} ${cx + bodyW + 2},${bodyCy + 4} Q${cx + bodyW * 0.55},${bodyCy - bodyRy * 0.35} ${cx},${bodyCy - bodyRy * 0.3} Q${cx - bodyW * 0.55},${bodyCy - bodyRy * 0.35} ${cx - bodyW - 2},${bodyCy + 4} Z" fill="${base}"/>` +
                `<ellipse cx="${cx}" cy="${headY - headRy * 0.75}" rx="${headRx * 0.55}" ry="${headRy * 0.35}" fill="${base}"/>` +
                `<ellipse cx="${cx}" cy="${headY + headRy * 0.45}" rx="${headRx * 0.45}" ry="${headRy * 0.4}" fill="${base}"/>`;
        case 'tricolor': return `<ellipse cx="${cx}" cy="${headY}" rx="${headRx}" ry="${headRy}" fill="${second}"/>` +
            `<ellipse cx="${cx}" cy="${bodyCy - bodyRy * 0.45}" rx="${bodyW * 1.05}" ry="${bodyRy * 0.6}" fill="${base}"/>` +
            `<ellipse cx="${cx}" cy="${headY - headRy * 0.65}" rx="${headRx * 0.75}" ry="${headRy * 0.4}" fill="${base}"/>` +
            chest(white) + blaze(white) + muzzle(white) + paws(white);
        case 'mask': return `<ellipse cx="${cx}" cy="${headY + headRy * 0.35}" rx="${headRx * 0.6}" ry="${headRy * 0.5}" fill="${second}"/>`;
        case 'tan': return `<ellipse cx="${cx}" cy="${headY + headRy * 0.5}" rx="${headRx * 0.45}" ry="${headRy * 0.35}" fill="${second}"/>` +
            `<circle cx="${cx - 12}" cy="${headY - 12}" r="3.5" fill="${second}"/><circle cx="${cx + 12}" cy="${headY - 12}" r="3.5" fill="${second}"/>` +
            chest(second) + paws(second);
        case 'patches': return spots(second === white ? base : second, 4, 11, 29) +
            `<ellipse cx="${cx + headRx * 0.4}" cy="${headY - headRy * 0.2}" rx="${headRx * 0.5}" ry="${headRy * 0.55}" fill="${base}"/>`;
        }
        return '';
    }

    // The tail, behind the body
    function tail(g, look, fill) {
        const { cx, bodyCy, bodyW, bodyRy } = g;
        const kind = oneOf(look.tail, ['long', 'plume', 'curled', 'otter', 'thin', 'stub', 'bushy'], 'long');
        const x = cx + bodyW * 0.7, y = bodyCy + bodyRy * 0.5;
        switch (kind) {
        case 'stub': return `<ellipse cx="${x + 6}" cy="${y - 6}" rx="7" ry="6" fill="${fill}"/>`;
        case 'curled': return `<path d="M${x},${y - 10} q24,-6 20,-30 q-4,-16 -18,-8 q-8,8 2,12" stroke="${fill}" stroke-width="11" fill="none" stroke-linecap="round"/>`;
        case 'otter': return `<path d="M${x},${y} q30,-4 34,-34" stroke="${fill}" stroke-width="13" fill="none" stroke-linecap="round"/>` +
            `<path d="M${x + 22},${y - 10} q8,-8 12,-24" stroke="${fill}" stroke-width="7" fill="none" stroke-linecap="round"/>`;
        case 'thin': return `<path d="M${x},${y} q34,0 30,-44" stroke="${fill}" stroke-width="5" fill="none" stroke-linecap="round"/>`;
        case 'plume':
        case 'bushy': {
            let out = `<path d="M${x},${y} q32,-2 30,-46" stroke="${fill}" stroke-width="${kind === 'bushy' ? 18 : 14}" fill="none" stroke-linecap="round"/>`;
            for (let i = 0; i < 4; i++) out += `<circle cx="${x + 18 + i * 3}" cy="${y - 10 - i * 11}" r="${kind === 'bushy' ? 10 : 8}" fill="${fill}"/>`;
            return out;
        }
        }
        return `<path d="M${x},${y} q34,0 32,-42" stroke="${fill}" stroke-width="9" fill="none" stroke-linecap="round"/>`;
    }

    // The face: eyes, nose, mouth and whiskers, for the pet's state
    function face(g, look, state, base) {
        const { cx, headY, headRx, headRy, dog, muzzle } = g;
        const out = [];
        const eyeY = headY - 3, eyeDX = headRx * 0.38;
        const iris = [color(look.eye, dog ? PALETTE.eye_brown : PALETTE.eye_green), color(look.eye2 || look.eye, dog ? PALETTE.eye_brown : PALETTE.eye_green)];
        const closed = ['sleeping', 'hibernating'].includes(state);
        const happy = ['happy', 'playing', 'social'].includes(state);
        const narrowed = state === 'sick' || state === 'reluctant';
        const ink = '#2b2220';

        // Muzzle
        if (dog && muzzle !== 'flat') {
            const ry = { short: 9, medium: 12, long: 15 }[muzzle];
            out.push(`<ellipse cx="${cx}" cy="${headY + headRy * 0.42 + (muzzle === 'long' ? 3 : 0)}" rx="${muzzle === 'long' ? 13 : 15}" ry="${ry}" fill="${shade(base, 0.25)}"/>`);
        } else if (!dog) {
            out.push(`<circle cx="${cx - 5}" cy="${headY + headRy * 0.38}" r="6" fill="${shade(base, 0.25)}"/><circle cx="${cx + 5}" cy="${headY + headRy * 0.38}" r="6" fill="${shade(base, 0.25)}"/>`);
        }

        // Eyes
        [-1, 1].forEach((side, i) => {
            const ex = cx + side * eyeDX;
            if (state === 'dead') {
                out.push(`<path d="M${ex - 4},${eyeY - 4} l8,8 M${ex + 4},${eyeY - 4} l-8,8" stroke="${ink}" stroke-width="2.2" stroke-linecap="round"/>`);
            } else if (closed) {
                out.push(`<path d="M${ex - 6},${eyeY} q6,5 12,0" stroke="${ink}" stroke-width="2.2" fill="none" stroke-linecap="round"/>`);
            } else if (happy) {
                out.push(`<path d="M${ex - 6},${eyeY + 2} q6,-8 12,0" stroke="${ink}" stroke-width="2.5" fill="none" stroke-linecap="round"/>`);
            } else {
                const r = dog ? 6 : 7;
                out.push(`<ellipse cx="${ex}" cy="${eyeY}" rx="${r}" ry="${narrowed ? r * 0.55 : r}" fill="${iris[i]}"/>`);
                out.push(dog ? `<circle cx="${ex}" cy="${eyeY}" r="${r * 0.55}" fill="${ink}"/>`
                    : `<ellipse cx="${ex}" cy="${eyeY}" rx="1.8" ry="${narrowed ? 3 : 5.5}" fill="${ink}"/>`);
                out.push(`<circle cx="${ex + 2}" cy="${eyeY - 2.5}" r="1.6" fill="#fff"/>`);
            }
        });

        // Nose and mouth
        const noseY = dog ? headY + headRy * (muzzle === 'flat' ? 0.25 : 0.32) : headY + headRy * 0.26;
        if (dog) {
            out.push(`<ellipse cx="${cx}" cy="${noseY}" rx="6.5" ry="4.5" fill="${ink}"/>`);
        } else {
            out.push(`<path d="M${cx - 4},${noseY - 2} L${cx + 4},${noseY - 2} L${cx},${noseY + 3} Z" fill="#e58f9e"/>`);
        }
        const mouthY = noseY + 6;
        if (happy || state === 'eating') {
            out.push(`<path d="M${cx - 8},${mouthY} q8,9 16,0 Z" fill="#7a2e2e"/>`);
            if (dog && happy) out.push(`<ellipse cx="${cx}" cy="${mouthY + 6}" rx="4.5" ry="5" fill="#ef7f8f"/>`);
        } else if (narrowed || state === 'dead') {
            out.push(`<path d="M${cx - 6},${mouthY + 3} q6,-5 12,0" stroke="${ink}" stroke-width="1.8" fill="none"/>`);
        } else {
            out.push(`<path d="M${cx - 6},${mouthY} q3,3 6,0 q3,3 6,0" stroke="${ink}" stroke-width="1.6" fill="none"/>`);
        }
        if (!dog) {
            for (const side of [-1, 1]) {
                out.push(`<path d="M${cx + side * 9},${noseY + 2} l${side * 18},-3 M${cx + side * 9},${noseY + 5} l${side * 18},2" stroke="rgba(60,50,45,0.45)" stroke-width="1"/>`);
            }
        }
        if (state === 'sick') {
            out.push(`<circle cx="${cx - eyeDX - 3}" cy="${eyeY + 10}" r="5" fill="#9ccc65" opacity="0.45"/><circle cx="${cx + eyeDX + 3}" cy="${eyeY + 10}" r="5" fill="#9ccc65" opacity="0.45"/>`);
        } else if (happy) {
            out.push(`<circle cx="${cx - eyeDX - 4}" cy="${eyeY + 10}" r="4.5" fill="#f48fb1" opacity="0.45"/><circle cx="${cx + eyeDX + 4}" cy="${eyeY + 10}" r="4.5" fill="#f48fb1" opacity="0.45"/>`);
        }
        return out.join('');
    }

    // Props around the pet for its state
    function props(g, state) {
        const { cx, ground } = g;
        const emoji = (text, x, y, size) => `<text x="${x}" y="${y}" font-size="${size}" text-anchor="middle">${text}</text>`;
        switch (state) {
        case 'eating': return `<ellipse cx="${cx}" cy="${ground}" rx="24" ry="6" fill="#b0bec5"/><path d="M${cx - 22},${ground - 8} h44 l-5,10 h-34 Z" fill="#78909c"/><ellipse cx="${cx}" cy="${ground - 8}" rx="18" ry="3.5" fill="#a1887f"/>`;
        case 'playing': return emoji('🧶', cx - 52, ground, 24);
        case 'sleeping': return emoji('💤', cx + 44, 52, 22);
        case 'sick': return emoji('🤒', cx - 50, 60, 18);
        case 'exploring': return emoji('🧭', cx - 52, ground - 4, 22);
        case 'social': return emoji('💕', cx + 46, 56, 20);
        case 'hibernating': return emoji('❄️', cx + 44, 52, 22);
        case 'reluctant': return emoji('💢', cx + 44, 56, 20);
        }
        return '';
    }

    // What the pet wears, at the right places on its body
    function outfit(g, worn) {
        const { cx, headY, headRy, bodyCy, bodyRy, bodyW } = g;
        const out = [];
        const slots = worn || {};
        const text = (c, x, y, size) => `<text x="${x}" y="${y}" font-size="${size}" text-anchor="middle"><title>${esc(c.name || '')}</title>${esc(c.icon || '')}</text>`;
        const neckY = headY + headRy - 2;
        const body = slots.body;
        if (body) {
            if (body.color && /^#[0-9a-fA-F]{6}$/.test(body.color)) {
                out.push(`<path d="M${cx - bodyW + 2},${bodyCy - bodyRy * 0.55} q${bodyW},-12 ${bodyW * 2 - 4},0 l2,${bodyRy * 1.05} q${-bodyW},10 ${-bodyW * 2},0 Z" fill="${body.color}" opacity="0.92"><title>${esc(body.name || '')}</title></path>`);
            } else {
                out.push(text(body, cx, bodyCy + 12, 30));
            }
        }
        const collar = slots.collar;
        if (collar) {
            const fill = collar.color && /^#[0-9a-fA-F]{6}$/.test(collar.color) ? collar.color : '#c62828';
            out.push(`<path d="M${cx - 24},${neckY - 2} q24,12 48,0" stroke="${fill}" stroke-width="6" fill="none" stroke-linecap="round"><title>${esc(collar.name || '')}</title></path>`);
            if (collar.icon && collar.icon !== '⭕') out.push(text(collar, cx, neckY + 16, 13));
        }
        if (slots.neck) {
            const n = slots.neck;
            if (n.icon === '🔻' && n.color && /^#[0-9a-fA-F]{6}$/.test(n.color)) {
                out.push(`<path d="M${cx - 22},${neckY - 3} L${cx + 22},${neckY - 3} L${cx},${neckY + 20} Z" fill="${n.color}"><title>${esc(n.name || '')}</title></path>`);
            } else {
                out.push(text(n, cx, neckY + 14, 22));
            }
        }
        if (slots.tag) out.push(text(slots.tag, cx, neckY + (collar ? 24 : 16), 12));
        if (slots.eyes) out.push(text(slots.eyes, cx, headY + 5, 30));
        if (slots.head) out.push(text(slots.head, cx, headY - headRy + 6, 34));
        return out.join('');
    }

    /**
     * Draws a pet
     * @param {object} opts
     * @param {string} opts.species - cat or dog
     * @param {object} opts.look - the pet's look (breed body, colors, pattern, eyes, scale)
     * @param {string} opts.state - idle, happy, eating, playing, sleeping, sick, exploring, social, hibernating, dead or reluctant
     * @param {object} opts.outfit - what it wears, by slot ({icon, color, name})
     * @param {number} opts.size - width and height in pixels
     * @param {string} opts.label - accessible label
     * @returns {string} SVG markup
     */
    function draw(opts) {
        const species = opts.species === 'dog' ? 'dog' : 'cat';
        const look = Object.assign({}, DEFAULTS[species], opts.look || {});
        const state = opts.state || 'idle';
        const g = geometry(species, look);
        const id = 'pet' + (++uid);

        const hairless = look.coat === 'hairless';
        let base = hairless ? shade(color(look.color, SKIN), 0.35) : color(look.color, '#9e9e9e');
        const second = look.color2 ? color(look.color2, '') : '';
        const white = PALETTE.white;
        let tailFill = base;
        if (look.pattern === 'white' || look.pattern === 'double_merle') base = white;
        if (look.pattern === 'van') {
            tailFill = base;
            base = white;
        }
        if (look.pattern === 'point' || look.pattern === 'mitted') tailFill = second || shade(base, -0.35);
        if (look.pattern === 'saddle') tailFill = second;
        const inner = hairless ? '#e7a3a0' : '#f3b6b9';
        const outline = shade(base === white ? '#cfcac0' : base, -0.3);

        // Smaller and larger breeds: 0.5 to 1.5 drawn as 60% to 100%
        const scale = Math.min(Math.max(0.6 + 0.4 * ((Number(look.scale) || 1) - 0.5), 0.55), 1.0);
        const transform = `translate(${100 - 100 * scale} ${g.ground - g.ground * scale}) scale(${scale})`;

        const parts = [];
        parts.push(`<defs><clipPath id="${id}c">${silhouette(g)}</clipPath></defs>`);
        parts.push(`<ellipse cx="100" cy="${g.ground + 2}" rx="${46 * scale}" ry="5" fill="rgba(0,0,0,0.12)"/>`);
        parts.push(`<g transform="${transform}"${state === 'dead' ? ' opacity="0.55"' : state === 'hibernating' ? ' opacity="0.85"' : ''}>`);
        parts.push(tail(g, look, tailFill));
        parts.push(ears(g, look, look.pattern === 'van' ? color(look.color, base) : (look.pattern === 'point' || look.pattern === 'mitted') ? (second || base) : base, inner, true));
        parts.push(coatTexture(g, look, base, false));
        parts.push(`<g stroke="${outline}" stroke-width="1.5">${silhouette(g).replace(/\/>/g, ` fill="${base}"/>`)}</g>`);
        parts.push(`<g clip-path="url(#${id}c)">${pattern(g, look, color(look.color, base), second, white)}`);
        if (look.pattern === 'van') {
            parts.push(`<ellipse cx="${g.cx}" cy="${g.headY - g.headRy * 0.55}" rx="${g.headRx}" ry="${g.headRy * 0.45}" fill="${color(look.color, '#d9822b')}"/>`);
        }
        parts.push(coatLines(g, look));
        parts.push('</g>');
        parts.push(coatTexture(g, look, look.pattern === 'tricolor' || look.pattern === 'tan' || look.pattern === 'bicolor' ? white : base, true));
        parts.push(ears(g, look, base, inner, false));
        parts.push(face(g, look, state, base === white ? '#e8e3d8' : base));
        parts.push(outfit(g, opts.outfit));
        parts.push('</g>');
        parts.push(props(g, state));
        if (state === 'hibernating') parts.push('<rect width="200" height="200" fill="#90caf9" opacity="0.15"/>');

        const size = opts.size || 120;
        return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 200 200" width="${size}" height="${size}" role="img" aria-label="${esc(opts.label || species)}">${parts.join('')}</svg>`;
    }

    return { draw };
})();
