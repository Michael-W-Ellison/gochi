// Help: the first-run tour, first-time tips, the Help panel, and the
// moments that play a sound. The household remembers which tips have been
// shown (only their names, from a fixed list), so each is shown once
// however many pets there are. Load after the other scripts: it adds to
// some of their functions.

const tourSteps = [
    ['#petScene', 'This is your pet. What it is doing and how it feels show here.'],
    ['.stats', 'These bars show health, energy, happiness, hunger and water. Care keeps them up.'],
    ['.actions', 'Care buttons: feed, give water, pet, play, groom and clean. Each one explains why it matters.'],
    ['.health-panel', 'Body: warning signs appear here before problems get serious. The 📖 Care guide explains each one.'],
    ['#checklist', 'The daily checklist and goals: small things to do each day, which earn Friendship Points.'],
    ['#friendship', 'Friendship Points, earned by caring, buy things in the Shop. Home Stars, earned by playing ' +
        'every day, buy house upgrades for all your pets.'],
    ['.activities', 'Activities: train tricks, explore, enter shows, play games and more.'],
    ['.save-row', 'Your pet is saved automatically. Pause stops time, and time also pauses when you are away.'],
];

const tips = {
    shop: ['🛒', 'The shop', 'Spend Friendship Points on equipment, decorations, outfits and supplies. Some equipment needs ' +
        'upkeep, such as a new filter, to keep working.'],
    house: ['⭐', 'The house', 'Home Stars, earned by caring and playing every day, buy house upgrades that help all your pets.'],
    breeding: ['💞', 'Breeding', 'Two healthy, happy adult pets of the same species can have a baby, which inherits its looks ' +
        'and tendencies from both parents.'],
    shows: ['🏅', 'Pet shows', 'Shows judge care, training and wellbeing, never looks. A pet rests a game day between shows.'],
    collection: ['🧺', 'The collection', 'Every souvenir your pets find goes in the household\'s collection. Completing a ' +
        'place\'s set earns Home Stars.'],
    family: ['👪', 'Family', 'A parent can give its young child Friendship Points, and when a pet dies its belongings can pass ' +
        'to its children.'],
    games: ['🎮', 'Games', 'Short games count as care. There are no scores: your pet shows how much it enjoyed playing.'],
    explore: ['🧭', 'Exploring', 'Trips find souvenirs and discover new places. Wilder places hold more and are riskier, and ' +
        'a hurt pet needs healing before it explores again.'],
};

const glossary = [
    ['Friendship Points ❤️', 'Earned by caring for your pet, at most once every 3 minutes for each kind of care, and spent in the shop.'],
    ['Home Stars ⭐', 'Earned by the household: playing each day, trips, shows, collections and achievements. They buy house upgrades.'],
    ['Body systems', 'The heart, lungs, kidneys, teeth, eyes and more, under Body. Each one shows how healthy it is.'],
    ['Warning signs', 'Signs your pet shows before a problem gets serious, such as a dull coat or bad breath.'],
    ['Tendencies', 'Health problems some breeds are prone to, such as dental disease in small dogs. Extra care helps.'],
    ['Upkeep', 'Some equipment needs regular care to keep working, such as a fountain\'s filter.'],
    ['Game days', 'Time in the game: one game day passes each real minute while you play.'],
];

let tipsShown = new Set(); // This session, before the server confirms

function tipSeen(id) {
    return tipsShown.has(id) || ((lastStatus && lastStatus.tips_seen) || []).includes(id);
}

function markTipSeen(id) {
    tipsShown.add(id);
    postJSON('/api/tips/seen', {id: id}).catch(console.error);
}

// showTip shows a first-time tip once
function showTip(id, replay) {
    const tip = tips[id];
    if (!tip || (!replay && tipSeen(id)) || popupShowing) return;
    popupShowing = 'tip:' + id;
    markTipSeen(id);
    showModal(tip[0], tip[1], tip[2], [['Got it', () => { popupShowing = null; }]]);
}

// The tour: one part of the screen at a time, with Next and Skip
let tourStep = -1;

function startTour() {
    if (popupShowing) return;
    popupShowing = 'tour';
    tourStep = -1;
    nextTourStep();
}

function nextTourStep() {
    document.querySelectorAll('.tour-highlight').forEach(e => e.classList.remove('tour-highlight'));
    tourStep++;
    let box = document.getElementById('tourBox');
    if (tourStep >= tourSteps.length) {
        if (box) box.remove();
        popupShowing = null;
        markTipSeen('tour');
        return;
    }
    if (!box) {
        box = el('div', 'tour-box');
        box.id = 'tourBox';
        box.setAttribute('role', 'dialog');
        box.setAttribute('aria-live', 'polite');
        document.body.append(box);
    }
    const [selector, text] = tourSteps[tourStep];
    const target = document.querySelector(selector);
    if (target) {
        target.classList.add('tour-highlight');
        target.scrollIntoView({block: 'center', behavior: 'smooth'});
    }
    box.replaceChildren(el('div', 'cloud-note', 'Tour ' + (tourStep + 1) + ' of ' + tourSteps.length), el('div', 'panel-text', text));
    const row = el('div', 'save-row');
    const next = cloudButton(row, tourStep === tourSteps.length - 1 ? 'Done' : 'Next', nextTourStep);
    cloudButton(row, 'Skip', () => { tourStep = tourSteps.length - 1; nextTourStep(); });
    box.append(row);
    next.focus();
}

// The Help panel
function openHelp() {
    panelMode = 'help';
    const p = panel();
    p.replaceChildren(panelHeader('❓ Help'));
    const row = el('div', 'save-row');
    cloudButton(row, 'Take the tour', () => { closePanel(); startTour(); });
    cloudButton(row, '📖 Care guide', () => openGuide());
    p.append(row);
    cloudText(p, 'Tips', 'section-title');
    const tipRow = el('div', 'cloud-row');
    Object.keys(tips).forEach(id => cloudButton(tipRow, tips[id][0] + ' ' + tips[id][1], () => showTip(id, true)));
    p.append(tipRow);
    cloudText(p, 'Words in the game', 'section-title');
    glossary.forEach(([word, meaning]) => {
        const item = el('div', 'panel-text');
        item.append(el('strong', '', word + ': '), meaning);
        p.append(item);
    });
    p.classList.remove('hidden');
}

// after adds to a page function: then runs after it, with its arguments
function after(name, then) {
    const original = window[name];
    if (typeof original !== 'function') return;
    window[name] = function (...args) {
        const result = original.apply(this, args);
        try { then(...args); } catch (e) { console.error(e); }
        return result;
    };
}

// First-time tips when a feature is first opened
[['openShop', 'shop'], ['renderHouse', 'house'], ['openBreeding', 'breeding'], ['openShows', 'shows'],
    ['openCollection', 'collection'], ['openFamily', 'family'], ['openGames', 'games'], ['explore', 'explore']]
    .forEach(([fn, id]) => after(fn, () => setTimeout(() => showTip(id), 400)));

// Sounds for care, points, warning signs, happenings and the tour
let lastCare = null, lastPoints = null, lastSigns = null;
after('interact', action => { lastCare = action; });
after('showResult', () => {
    const sounds = {feed: 'munch', drink: 'lap', pet: null, play: 'squeak', groom: 'rustle', clean: 'rustle', walk: null};
    if (lastCare === 'pet' || lastCare === 'walk') petSound('happy');
    else if (sounds[lastCare]) playSound(sounds[lastCare]);
    lastCare = null;
});
after('updateActivities', status => {
    renderTheme(status);
    const points = status.friendship_points, signs = (status.symptoms || []).filter(s => !s.good).length;
    if (lastPoints !== null && points > lastPoints) playSound('chime');
    if (lastSigns !== null && signs > lastSigns) playSound('alert');
    lastPoints = points;
    lastSigns = signs;
    if (status.is_alive && !tipSeen('tour') && !popupShowing && tourStep < 0) setTimeout(startTour, 800);
});
after('showHappening', h => { if (h && popupShowing === 'happening:' + h.id) playSound('bell'); });
after('openGuide', () => playSound('page'));
