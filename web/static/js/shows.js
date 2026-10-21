// Pet shows: choosing a show and level (and a trick routine), and the
// results. Uses panel helpers from the page and el() from shop.js.

const levelNames = {1: 'Novice', 2: 'Open', 3: 'Champion'};
const placeNames = ['1st', '2nd', '3rd'];
const rosetteIcons = ['🥇', '🥈', '🥉'];

function openShows() {
    panelMode = 'shows';
    fetch('/api/shows').then(r => r.json()).then(data => {
        if (data.success) renderShows(data.data);
    }).catch(console.error);
}

function renderShows(data) {
    const p = panel();
    p.replaceChildren(panelHeader('🏅 Pet shows'));
    cloudText(p, 'Shows judge care, training and wellbeing — never looks — so any pet can win. A show tires your pet a ' +
        'little, so it rests a game day before the next.', 'panel-text cloud-note');
    if (data.titles.includes('grand_champion')) cloudText(p, '👑 Grand Champion', 'panel-text');
    if (data.ribbons.length) {
        const won = data.ribbons.slice(-8).map(r => rosetteIcons[r.place - 1] + ' ' + levelNames[r.level] + ' ' +
            (data.shows.find(s => s.id === r.show) || {name: r.show}).name);
        cloudText(p, 'Ribbons: ' + won.join(' · '), 'panel-text');
    }
    const known = data.tricks.filter(t => t.learned && !t.special);
    data.shows.forEach(show => {
        const box = el('div', 'show-box');
        box.append(el('div', 'show-title', show.icon + ' ' + show.name + (show.fee ? ' · ' + show.fee + ' ❤️ to enter' : ' · free once a day')));
        box.append(el('div', 'cloud-note', show.judged + '.'));
        if (show.unavailable) {
            box.append(el('div', 'cloud-note', show.unavailable));
            p.append(box);
            return;
        }
        const row = el('div', 'show-row');
        const level = el('select', 'show-level');
        level.setAttribute('aria-label', 'Level');
        show.levels.forEach(l => {
            const o = el('option', '', levelNames[l]);
            o.value = l;
            level.append(o);
        });
        level.value = show.levels[show.levels.length - 1];
        row.append(level);
        const routine = [];
        if (show.id === 'tricks') {
            if (known.length < 3) {
                box.append(el('div', 'cloud-note', 'Needs three tricks learned well (rolling over and playing dead don\'t count for cats).'));
                p.append(box);
                return;
            }
            for (let i = 0; i < 3; i++) {
                const pick = el('select', 'show-trick');
                pick.setAttribute('aria-label', 'Trick ' + (i + 1));
                known.forEach(t => {
                    const o = el('option', '', t.icon + ' ' + t.name);
                    o.value = t.id;
                    pick.append(o);
                });
                pick.value = known[Math.min(i, known.length - 1)].id;
                routine.push(pick);
                row.append(pick);
            }
        }
        cloudButton(row, 'Enter', () => enterShow(show.id, Number(level.value), routine.map(s => s.value)));
        box.append(row);
        p.append(box);
    });
    p.classList.remove('hidden');
}

function enterShow(show, level, routine) {
    postJSON('/api/shows/enter', {show, level, routine}).then(data => {
        if (!data.success) {
            showPointsToast(data.error || 'Something went wrong');
            return;
        }
        const r = data.data;
        const p = panel();
        p.replaceChildren(panelHeader(r.show.icon + ' ' + levelNames[r.level] + ' ' + r.show.name));
        cloudText(p, r.text, 'panel-text');
        const table = el('ol', 'show-results');
        r.entrants.forEach(e => {
            const item = el('li', e.you ? 'you' : '', e.name + ' the ' + e.breed + ' — ' + e.score);
            table.append(item);
        });
        p.append(table);
        if (r.points || r.stars) cloudText(p, '+' + r.points + ' ❤️' + (r.stars ? ' +' + r.stars + ' ⭐' : ''), 'panel-text');
        cloudText(p, '💡 ' + r.lesson, 'panel-text cloud-note');
        const row = el('div', '');
        cloudButton(row, 'Back to shows', openShows);
        p.append(row);
        fetchStatus();
    }).catch(console.error);
}
