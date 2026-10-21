// Mini-games: fetch (dogs), the feather wand (cats), the puzzle feeder,
// the scent trail and gentle grooming. Each works with a mouse, touch or
// the keyboard. There are no scores: the game tells the server how well
// it went (0 to 1), and the result says how the pet enjoyed it. Relaxed
// mode removes time pressure. Uses panel helpers from the page and el()
// from shop.js.

// relaxedMode is remembered on this device only
function relaxedMode() {
    try { return localStorage.getItem('gochiRelaxedGames') === '1'; } catch (e) { return false; }
}

function setRelaxedMode(on) {
    try { localStorage.setItem('gochiRelaxedGames', on ? '1' : '0'); } catch (e) { /* Not remembered */ }
}

// gameRunning reports whether a game's panel is still open
function gameRunning(box) {
    return panelMode === 'game' && box.isConnected;
}

function petName() {
    return (lastStatus && lastStatus.name) || 'Your pet';
}

function openGames() {
    panelMode = 'games';
    fetch('/api/games').then(r => r.json()).then(data => {
        if (data.success) renderGames(data.data);
        else showPointsToast(data.error || 'Something went wrong');
    }).catch(console.error);
}

function renderGames(games) {
    const p = panel();
    p.replaceChildren(panelHeader('🎮 Games'));
    cloudText(p, 'Short games to play with ' + petName() + '. Each counts as play, enrichment or grooming.',
        'panel-text cloud-note');
    const relaxed = el('label', 'panel-text');
    const box = el('input');
    box.type = 'checkbox';
    box.checked = relaxedMode();
    box.onchange = () => setRelaxedMode(box.checked);
    relaxed.append(box, ' Relaxed mode: no time limits, and a slower marker in fetch');
    p.append(relaxed);
    games.forEach(g => {
        const row = el('div', 'show-box');
        row.append(el('div', 'show-title', g.icon + ' ' + g.name));
        row.append(el('div', 'cloud-note', g.unavailable || g.description));
        if (!g.unavailable) {
            const r = el('div', '');
            cloudButton(r, 'Play', () => startGame(g));
            row.append(r);
        }
        p.append(row);
    });
    p.classList.remove('hidden');
}

function startGame(game) {
    panelMode = 'game';
    const p = panel();
    p.replaceChildren(panelHeader(game.icon + ' ' + game.name));
    cloudText(p, game.description, 'panel-text cloud-note');
    const box = el('div', 'game-box');
    p.append(box);
    const finish = quality => finishGame(game, quality);
    switch (game.id) {
    case 'fetch': playFetch(box, finish); break;
    case 'wand': playWand(box, finish); break;
    case 'puzzle': playPuzzle(box, game.tiles || 4, finish); break;
    case 'scent': playScent(box, finish); break;
    case 'grooming': playGrooming(box, game.regions || [], finish); break;
    }
    p.classList.remove('hidden');
}

function finishGame(game, quality) {
    postJSON('/api/games/finish', {game: game.id, quality: Math.max(0, Math.min(1, quality))}).then(data => {
        const p = panel();
        p.replaceChildren(panelHeader(game.icon + ' ' + game.name));
        panelMode = 'games';
        if (!data.success) {
            cloudText(p, data.error || 'Something went wrong', 'panel-text');
        } else {
            const r = data.data;
            const faces = {loved: '😻', enjoyed: '😊', some_fun: '🙂', bored: '😐'};
            cloudText(p, (faces[r.enjoyment] || '') + ' ' + r.text, 'panel-text');
            if (r.points_earned) cloudText(p, '+' + r.points_earned + ' ❤️', 'panel-text');
            cloudText(p, '💡 ' + r.game_lesson, 'panel-text cloud-note');
        }
        const row = el('div', '');
        cloudButton(row, 'Play again', () => startGame(game));
        cloudButton(row, 'Other games', openGames);
        p.append(row);
        p.classList.remove('hidden');
        fetchStatus();
    }).catch(console.error);
}

// Fetch: throw when the marker is in the green. Five throws.
function playFetch(box, finish) {
    const throws = 5;
    let thrown = 0, good = 0, pos = 0, dir = 1;
    const speed = relaxedMode() ? 0.6 : 1.4;
    const meter = el('div', 'game-meter');
    const zone = el('div', 'game-zone');
    const marker = el('div', 'game-marker');
    meter.append(zone, marker);
    const field = el('div', 'game-field', '🐕 ready...');
    const status = el('div', 'cloud-note', 'Throw 1 of ' + throws);
    const row = el('div', '');
    const btn = cloudButton(row, 'Throw!', () => {
        if (thrown >= throws) return;
        const inGreen = pos >= 55 && pos <= 80;
        thrown++;
        if (inGreen) good++;
        field.textContent = inGreen ? '🎾 ... 🐕💨 A great throw into the open, and back it comes!' :
            pos < 55 ? '🎾 A short throw; ' + petName() + ' trots over, unimpressed.' :
            '🎾 Too far, into the bushes; ' + petName() + ' takes ages to find it.';
        status.textContent = thrown < throws ? 'Throw ' + (thrown + 1) + ' of ' + throws : 'That\'s enough for now.';
        if (thrown >= throws) {
            btn.disabled = true;
            setTimeout(() => { if (gameRunning(box)) finish(good / throws); }, 900);
        }
    });
    box.append(meter, field, status, row);
    const step = () => {
        if (!gameRunning(box) || thrown >= throws) return;
        pos += dir * speed;
        if (pos >= 100 || pos <= 0) dir = -dir;
        marker.style.left = pos + '%';
        requestAnimationFrame(step);
    };
    requestAnimationFrame(step);
    btn.focus();
}

// Feather wand: move like prey (quick dashes and slow creeping), then let
// the cat catch it
function playWand(box, finish) {
    const area = el('div', 'game-area');
    area.tabIndex = 0;
    area.setAttribute('aria-label', 'Play area: move the wand with the mouse, a finger or the arrow keys');
    const cat = el('div', 'game-sprite', '🐈');
    const wand = el('div', 'game-sprite', '🪶');
    area.append(cat, wand);
    const status = el('div', 'cloud-note', 'Move the wand: quick dashes and slow, creeping moves.');
    const row = el('div', '');
    box.append(area, status, row);
    let wx = 200, wy = 60, cx = 20, cy = 110, fast = 0, slow = 0, last = {x: wx, y: wy, t: performance.now()};
    let done = false;
    const place = () => {
        wand.style.left = wx + 'px'; wand.style.top = wy + 'px';
        cat.style.left = cx + 'px'; cat.style.top = cy + 'px';
    };
    const moveTo = (x, y) => {
        wx = Math.max(0, Math.min(area.clientWidth - 24, x));
        wy = Math.max(0, Math.min(area.clientHeight - 24, y));
        place();
    };
    area.addEventListener('pointermove', e => {
        const r = area.getBoundingClientRect();
        moveTo(e.clientX - r.left - 12, e.clientY - r.top - 12);
    });
    area.addEventListener('keydown', e => {
        const steps = {ArrowLeft: [-1, 0], ArrowRight: [1, 0], ArrowUp: [0, -1], ArrowDown: [0, 1]}[e.key];
        if (!steps) return;
        e.preventDefault();
        const size = e.shiftKey ? 60 : 8; // Shift for a quick dash
        moveTo(wx + steps[0] * size, wy + steps[1] * size);
    });
    const end = caught => {
        if (done) return;
        done = true;
        const variety = Math.min(fast / 6, 1) * 0.3 + Math.min(slow / 6, 1) * 0.3;
        status.textContent = caught ? petName() + ' pounces and catches the feather! 🎉' : 'The feather stopped before ' +
            petName() + ' could catch it.';
        setTimeout(() => { if (gameRunning(box)) finish(variety + (caught ? 0.4 : 0)); }, 900);
    };
    cloudButton(row, 'Let it catch the feather', () => end(true));
    const started = performance.now();
    const tick = () => {
        if (!gameRunning(box) || done) return;
        const now = performance.now();
        if (now - last.t >= 250) {
            const v = Math.hypot(wx - last.x, wy - last.y) / ((now - last.t) / 1000);
            if (v > 250) fast++;
            else if (v > 10 && v < 80) slow++;
            last = {x: wx, y: wy, t: now};
        }
        // The cat stalks the feather
        cx += (wx - cx) * 0.02;
        cy += (wy - cy) * 0.02;
        place();
        if (!relaxedMode() && now - started > 30000) end(false);
        else requestAnimationFrame(tick);
    };
    place();
    requestAnimationFrame(tick);
    area.focus();
}

// Puzzle feeder: lift tiles to find the kibble; the pet loses patience
// with too many empty tiles
function playPuzzle(box, tiles, finish) {
    const kibble = Math.ceil(tiles / 2);
    const hidden = new Set();
    while (hidden.size < kibble) hidden.add(Math.floor(Math.random() * tiles));
    let found = 0, patience = 2, over = false;
    const grid = el('div', 'game-tiles');
    const status = el('div', 'cloud-note', petName() + ' sniffs the puzzle. ' + kibble + ' pieces of kibble are hidden.');
    for (let i = 0; i < tiles; i++) {
        const t = el('button', 'game-tile', '🟫');
        t.setAttribute('aria-label', 'Tile ' + (i + 1));
        t.onclick = () => {
            if (over) return;
            t.disabled = true;
            if (hidden.has(i)) {
                found++;
                t.textContent = '🍪';
                status.textContent = petName() + ' gobbles up the kibble! ' + (kibble - found) + ' left.';
            } else {
                patience--;
                t.textContent = '·';
                status.textContent = 'Nothing there. ' + (patience > 0 ? petName() + ' noses at the other tiles.' :
                    petName() + ' is losing patience.');
            }
            if (found === kibble || patience < 0) {
                over = true;
                setTimeout(() => { if (gameRunning(box)) finish(found / kibble); }, 900);
            }
        };
        grid.append(t);
    }
    // The pet's nose gives a hint now and then
    const hint = el('div', 'cloud-note');
    const row = el('div', '');
    cloudButton(row, 'Watch ' + petName() + '\'s nose', () => {
        const left = [...hidden].filter(i => !grid.children[i].disabled);
        if (!left.length || over) return;
        const i = left[Math.floor(Math.random() * left.length)];
        hint.textContent = '👃 ' + petName() + ' sniffs hard at tile ' + (i + 1) + '.';
    });
    box.append(grid, status, hint, row);
}

// Scent trail: three treats hidden around the house; the pet's tail and
// ears say warmer or colder
function playScent(box, finish) {
    const rooms = ['🍳 Kitchen', '🚪 Hall', '🛋️ Living room', '🛏️ Bedroom', '🛁 Bathroom', '🌿 Back door'];
    const treats = 3;
    let found = 0, guesses = 0, treat = Math.floor(Math.random() * rooms.length), lastGuess = null;
    const status = el('div', 'panel-text', petName() + ' is sniffing the air. Where is the first treat?');
    const grid = el('div', 'game-tiles');
    rooms.forEach((name, i) => {
        const b = el('button', 'game-tile wide', name);
        b.onclick = () => {
            if (found >= treats) return;
            guesses++;
            if (i === treat) {
                found++;
                lastGuess = null;
                status.textContent = '🦴 Found it in the ' + name.split(' ').slice(1).join(' ').toLowerCase() + '! ' +
                    (found < treats ? 'On to the next one.' : 'All three found.');
                if (found >= treats) {
                    setTimeout(() => { if (gameRunning(box)) finish(Math.min(1, (treats * 2) / guesses)); }, 900);
                } else {
                    let next;
                    do { next = Math.floor(Math.random() * rooms.length); } while (next === treat);
                    treat = next;
                }
                return;
            }
            const distance = Math.abs(i - treat);
            if (lastGuess === null) status.textContent = 'Not here. ' + (distance <= 1 ?
                petName() + '\'s tail wags faster: it\'s close.' : petName() + '\'s ears droop: it\'s far away.');
            else status.textContent = 'Not here. ' + (distance < Math.abs(lastGuess - treat) ?
                petName() + '\'s tail wags faster: warmer!' : distance > Math.abs(lastGuess - treat) ?
                petName() + '\'s ears droop: colder.' : petName() + ' sniffs, unsure: about the same.');
            lastGuess = i;
        };
        grid.append(b);
    });
    box.append(grid, status);
}

// Gentle grooming: brush each part of the coat the way the fur lies, and
// gently where the pet is sensitive
function playGrooming(box, regions, finish) {
    const arrows = ['⬅️', '➡️', '⬆️', '⬇️'];
    let done = 0, right = 0;
    const status = el('div', 'panel-text', 'Brush each part the way its fur lies.');
    const gentle = el('label', 'panel-text');
    const soft = el('input');
    soft.type = 'checkbox';
    gentle.append(soft, ' Brush gently');
    box.append(gentle);
    regions.forEach(region => {
        const lies = Math.floor(Math.random() * arrows.length);
        const row = el('div', 'trick-row');
        const label = el('span', 'trick-name', region.name);
        const info = el('span', 'cloud-note', 'Fur lies ' + arrows[lies] + (region.sensitive ? ' · ' + petName() +
            ' watches your hand warily' : ''));
        const buttons = el('div', '');
        arrows.forEach((a, i) => {
            const b = cloudButton(buttons, a, () => {
                [...buttons.children].forEach(c => { c.disabled = true; });
                done++;
                const withFur = i === lies;
                const flinch = region.sensitive && !soft.checked;
                if (withFur && !flinch) right++;
                status.textContent = flinch ? '😾 ' + petName() + ' flinches: that spot needs a gentle touch.' :
                    withFur ? '😌 Smooth strokes; ' + petName() + ' relaxes.' :
                    '😣 Against the fur: that pulls.';
                if (done === regions.length) {
                    setTimeout(() => { if (gameRunning(box)) finish(right / regions.length); }, 900);
                }
            });
            b.setAttribute('aria-label', 'Brush ' + region.name + ' ' + ['left', 'right', 'up', 'down'][i]);
        });
        row.append(label, info, buttons);
        box.append(row);
    });
    box.append(status);
}
