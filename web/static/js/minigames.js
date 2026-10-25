// Mini-games: fetch (dogs), the feather wand (cats), the puzzle feeder
// (lifting tiles, a memory board, a sniff mat or a cup shuffle, by level),
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
    if (petAsleep()) return;
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
    case 'puzzle': playPuzzle(box, game.puzzle, finish); break;
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
            if (r.note) cloudText(p, '🧩 ' + r.note, 'panel-text');
            cloudText(p, '💡 ' + r.game_lesson, 'panel-text cloud-note');
        }
        const row = el('div', '');
        // Again with the game as it is now (the puzzle's level may have changed)
        cloudButton(row, 'Play again', () => fetch('/api/games').then(r => r.json()).then(d => {
            const now = d.success && d.data.find(g => g.id === game.id);
            if (now && !now.unavailable) startGame(now);
            else openGames();
        }).catch(console.error));
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
        playSound('throw');
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
        if (caught) playSound('catch');
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

// Puzzle feeder: one of the feeders its level opens, picked each game
// (see PuzzleSetup in internal/core/minigames.go). The pet's nose gives a
// few hints: at low levels which spot, later only which row or side.
const puzzleFeeders = {
    tiles: ['Lift the tiles', 'Lift the tiles to find the kibble.'],
    memory: ['Memory board', 'Watch where the kibble is before the board is covered, then find it.'],
    sniff: ['Sniff mat', 'An empty spot shows how much kibble is next to it, also diagonally. Use the numbers to find the rest.'],
    cups: ['Cup shuffle', 'Watch the cup with the kibble while the cups are swapped, then pick it.'],
};

function playPuzzle(box, setup, finish) {
    setup = setup || {level: 1, kinds: ['tiles'], tiles: 4, kibble: 2, misses: 3, sniffs: 3, exact: true};
    const kinds = (setup.kinds || []).filter(k => puzzleFeeders[k]);
    const kind = kinds.length ? kinds[Math.floor(Math.random() * kinds.length)] : 'tiles';
    box.append(el('div', 'show-title', 'Level ' + setup.level + ' · ' + puzzleFeeders[kind][0]),
        el('div', 'cloud-note', puzzleFeeders[kind][1]));
    if (kind === 'cups') playCups(box, setup, finish);
    else playBoard(box, setup, kind, finish);
}

// sniffButton adds the button for the pet's nose: hint(text) is called
// with what it shows, while sniffs are left
function sniffButton(box, sniffs, hint) {
    const text = el('div', 'cloud-note');
    const row = el('div', '');
    let left = sniffs;
    const label = () => 'Watch ' + petName() + '\'s nose (' + left + ' left)';
    const btn = cloudButton(row, label(), () => {
        const said = hint();
        if (!said) return;
        text.textContent = '👃 ' + said;
        left--;
        btn.textContent = label();
        btn.disabled = left <= 0;
    });
    btn.disabled = left <= 0;
    box.append(text, row);
    return btn;
}

// playBoard plays the feeders on a board of spots: lifting tiles, the
// memory board and the sniff mat
function playBoard(box, setup, kind, finish) {
    const tiles = setup.tiles, kibble = Math.min(setup.kibble, tiles - 1);
    const cols = tiles <= 6 ? 3 : 4;
    const hidden = new Set();
    while (hidden.size < kibble) hidden.add(Math.floor(Math.random() * tiles));
    // The sniff mat's numbers make wrong picks part of the game
    let misses = setup.misses + (kind === 'sniff' ? 2 : 0);
    let found = 0, over = kind === 'memory'; // The memory board waits until it is covered
    const grid = el('div', 'game-tiles');
    grid.style.gridTemplateColumns = 'repeat(' + cols + ', 1fr)';
    const status = el('div', 'cloud-note', petName() + ' sniffs the puzzle. ' + kibble + ' pieces of kibble are hidden.');
    const near = i => {
        const r = Math.floor(i / cols), c = i % cols;
        let n = 0;
        hidden.forEach(j => {
            const jr = Math.floor(j / cols), jc = j % cols;
            if (j !== i && Math.abs(jr - r) <= 1 && Math.abs(jc - c) <= 1) n++;
        });
        return n;
    };
    for (let i = 0; i < tiles; i++) {
        const t = el('button', 'game-tile', '🟫');
        t.setAttribute('aria-label', 'Spot ' + (i + 1) + ', row ' + (Math.floor(i / cols) + 1));
        t.onclick = () => {
            if (over) return;
            t.disabled = true;
            playSound('rustle');
            if (hidden.has(i)) {
                found++;
                t.textContent = '🍪';
                status.textContent = petName() + ' gobbles up the kibble! ' + (kibble - found) + ' left.';
            } else {
                misses--;
                t.textContent = kind === 'sniff' ? String(near(i) || '·') : '·';
                status.textContent = 'Nothing there. ' + (misses >= 0 ? petName() + ' noses at the other spots.' :
                    petName() + ' is losing patience.');
            }
            if (found === kibble || misses < 0) {
                over = true;
                setTimeout(() => { if (gameRunning(box)) finish(found / kibble); }, 900);
            }
        };
        grid.append(t);
    }
    box.append(grid, status);
    sniffButton(box, setup.sniffs, () => {
        const left = [...hidden].filter(i => !grid.children[i].disabled);
        if (!left.length || over) return '';
        const i = left[Math.floor(Math.random() * left.length)];
        return setup.exact ? petName() + ' sniffs hard at spot ' + (i + 1) + '.' :
            petName() + ' sniffs along row ' + (Math.floor(i / cols) + 1) + '.';
    });
    if (kind === 'memory') {
        [...grid.children].forEach((t, i) => { t.disabled = true; if (hidden.has(i)) t.textContent = '🍪'; });
        status.textContent = 'Watch where the kibble is!';
        const seconds = setup.show_seconds * (relaxedMode() ? 2 : 1);
        setTimeout(() => {
            if (!gameRunning(box)) return;
            [...grid.children].forEach(t => { t.disabled = false; t.textContent = '🟫'; });
            status.textContent = 'Now find the ' + kibble + ' pieces of kibble.';
            over = false;
        }, seconds * 1000);
    }
}

// playCups plays the cup shuffle: a round each time, the kibble shown
// under a cup, the cups swapped, then the player picks
function playCups(box, setup, finish) {
    const count = setup.cups || 3, rounds = setup.rounds || 3;
    const swapMs = (setup.swap_seconds || 0.6) * 1000 * (relaxedMode() ? 1.6 : 1);
    const grid = el('div', 'game-tiles');
    grid.style.gridTemplateColumns = 'repeat(' + count + ', 1fr)';
    const status = el('div', 'cloud-note');
    let round = 0, wins = 0, under = 0, picking = false;
    const cups = [];
    for (let i = 0; i < count; i++) {
        const c = el('button', 'game-tile', '🥣');
        c.setAttribute('aria-label', 'Cup ' + (i + 1));
        c.disabled = true;
        c.onclick = () => {
            if (!picking) return;
            picking = false;
            cups.forEach(x => { x.disabled = true; });
            const pos = cups.indexOf(c);
            cups[under].textContent = '🍪';
            if (pos === under) {
                wins++;
                playSound('rustle');
                status.textContent = 'Found it! ' + petName() + ' gobbles up the kibble.';
            } else {
                c.textContent = '·';
                status.textContent = 'Not that one. The kibble was under cup ' + (under + 1) + '.';
            }
            setTimeout(nextRound, 1200);
        };
        cups.push(c);
        grid.append(c);
    }
    box.append(grid, status);
    // A hint only while the cups can be picked
    sniffButton(box, setup.sniffs, () => {
        if (!picking) return '';
        return setup.exact ? petName() + ' sniffs hard at cup ' + (under + 1) + '.' :
            petName() + ' sniffs at the ' + (under < count / 2 ? 'left' : 'right') + ' side.';
    });
    // Swap two cups' places on the board, showing which two
    const swap = (a, b, then) => {
        cups[a].classList.add('swapping');
        cups[b].classList.add('swapping');
        setTimeout(() => {
            if (!gameRunning(box)) return;
            cups[a].classList.remove('swapping');
            cups[b].classList.remove('swapping');
            [cups[a], cups[b]] = [cups[b], cups[a]];
            grid.replaceChildren(...cups);
            if (under === a) under = b;
            else if (under === b) under = a;
            setTimeout(then, swapMs / 3);
        }, swapMs);
    };
    function nextRound() {
        if (!gameRunning(box)) return;
        if (round === rounds) {
            finish(wins / rounds);
            return;
        }
        round++;
        cups.forEach(c => { c.textContent = '🥣'; });
        under = Math.floor(Math.random() * count);
        cups[under].textContent = '🍪';
        status.textContent = 'Round ' + round + ' of ' + rounds + ': watch the kibble!';
        let left = setup.swaps || 4;
        const step = () => {
            if (!gameRunning(box)) return;
            if (left-- <= 0) {
                status.textContent = 'Round ' + round + ' of ' + rounds + ': which cup has the kibble?';
                cups.forEach(c => { c.disabled = false; });
                picking = true;
                return;
            }
            const a = Math.floor(Math.random() * count);
            const b = (a + 1 + Math.floor(Math.random() * (count - 1))) % count;
            swap(a, b, step);
        };
        setTimeout(() => {
            if (!gameRunning(box)) return;
            cups[under].textContent = '🥣';
            step();
        }, 1200 * (relaxedMode() ? 1.5 : 1));
    }
    nextRound();
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
                playSound('catch');
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
                if (flinch) petSound('sad');
                else playSound('rustle');
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
