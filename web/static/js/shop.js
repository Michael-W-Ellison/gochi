/**
 * The shop (equipment, decorations, wardrobe, pantry and house), how the
 * pet and its home look, the family panel, the hemisphere setting and
 * pop-ups. Uses helpers from the page's own script: postJSON, panel,
 * panelHeader, cloudText, cloudButton, showPointsToast, formatPoints,
 * speciesInfo and closePanel.
 */

let shopTab = 'equipment';
let shopData = null;
let suppliesData = null;
let homeData = null;

const shopTabs = [
    ['equipment', '🧰 Equipment'], ['decor', '🖼️ Decorations'], ['cosmetics', '🎀 Wardrobe'],
    ['pantry', '🥫 Pantry'], ['house', '⭐ House'],
];
const equipmentSections = [
    ['food_bowl', 'Food'], ['water_bowl', 'Water'], ['toy', 'Toys (kept)'], ['bed', 'Beds'], ['brush', 'Grooming'],
    ['training', 'Training'], ['home', 'Hygiene'], ['comfort', 'Comfort and weather'], ['health', 'Safety and health'],
    ['enrichment', 'Things to do'], ['old_age', 'Old age'], ['seasonal', 'In season now'],
];
const slotNames = {head: 'Head', eyes: 'Eyes', neck: 'Neck', collar: 'Collar', tag: 'Tag', body: 'Body'};
const slotOrder = ['head', 'eyes', 'neck', 'collar', 'tag', 'body'];
const supplySections = [
    ['food', 'Food (used up)'], ['treat', 'Treats (used up)'], ['toy', 'Toys (wear out)'],
    ['care', 'Care and upkeep (used up)'], ['health', 'Health (used up)'],
];
const seasonNames = {Spring: '🌸 Spring', Summer: '☀️ Summer', Fall: '🍂 Autumn', Winter: '❄️ Winter'};

function el(tag, className, text) {
    const e = document.createElement(tag);
    if (className) e.className = className;
    if (text !== undefined) e.textContent = text;
    return e;
}

function openShop() {
    shopping = true;
    document.getElementById('app').classList.add('hidden');
    document.getElementById('shopView').classList.remove('hidden');
    document.getElementById('shopMsg').textContent = '';
    askHemisphereOnce().then(loadShop);
}

function closeShop() {
    shopping = false;
    document.getElementById('shopView').classList.add('hidden');
    document.getElementById('app').classList.remove('hidden');
    fetchStatus();
}

function loadShop() {
    return Promise.all([
        fetch('/api/shop').then(r => r.json()),
        fetch('/api/supplies').then(r => r.json()),
        fetch('/api/home').then(r => r.json()).catch(() => ({success: false})),
    ]).then(([shop, supplies, home]) => {
        if (shop.success) shopData = shop.data;
        if (supplies.success) suppliesData = supplies.data;
        homeData = home.success ? home.data : null;
        renderShopView();
    }).catch(err => console.error('Shop error:', err));
}

function renderShopView() {
    if (!shopData) return;
    const points = document.getElementById('shopPoints');
    points.textContent = formatPoints(shopData.friendship_points) + (homeData ? ' · ⭐ ' + homeData.stars + ' Home Stars' : '');

    const list = document.getElementById('shopList');
    list.replaceChildren();
    const tabs = el('div', 'shop-tabs');
    tabs.setAttribute('role', 'tablist');
    shopTabs.forEach(([id, title]) => {
        if (id === 'house' && !homeData) return;
        const tab = el('button', 'shop-tab' + (shopTab === id ? ' active' : ''), title);
        tab.setAttribute('role', 'tab');
        tab.setAttribute('aria-selected', shopTab === id ? 'true' : 'false');
        tab.onclick = () => { shopTab = id; renderShopView(); };
        tabs.append(tab);
    });
    list.append(tabs);
    list.append(occasionBanner());

    ({equipment: renderEquipment, decor: renderDecor, cosmetics: renderWardrobe, pantry: renderPantry,
        house: renderHouse}[shopTab] || renderEquipment)(list);
}

// What's on sale for a limited time: the season and any holidays
function occasionBanner() {
    const banner = el('div', 'occasion-banner');
    const parts = [(seasonNames[shopData.season] || shopData.season) + ' items are in the shop this season.'];
    (shopData.holidays || []).forEach(h => parts.push(h.icon + ' ' + h.name + ' items are on sale now.'));
    if ((shopData.holidays || []).length) {
        parts.push(shopData.holiday_bought ? 'You have bought your holiday item for this visit: come back next time for another.' :
            'One holiday item per visit.');
    }
    banner.textContent = parts.join(' ');
    return banner;
}

function shopRow(icon, title, badges, lines, button, className) {
    const row = el('div', 'shop-item' + (className ? ' ' + className : ''));
    const iconBox = el('div', 'shop-icon', icon);
    const info = el('div', 'shop-info');
    const name = el('div', 'shop-name', title);
    badges.filter(Boolean).forEach(([text, cls]) => name.append(el('span', 'badge ' + cls, text)));
    info.append(name);
    lines.filter(Boolean).forEach(([text, cls]) => info.append(el('div', cls || 'shop-desc', text)));
    row.append(iconBox, info);
    if (button) row.append(button);
    return row;
}

function priceButton(price, symbol, affordable, onclick) {
    const btn = el('button', 'save-btn', symbol + ' ' + price);
    btn.disabled = !affordable;
    btn.onclick = onclick;
    return btn;
}

function lockedButton(unlockAt) {
    const btn = el('button', 'save-btn', '🔒 ' + unlockAt);
    btn.title = 'Unlocks once your pet has earned ' + unlockAt + ' Friendship Points in all';
    btn.disabled = true;
    return btn;
}

function ownedButton(text) {
    const btn = el('button', 'save-btn', text || 'Owned');
    btn.disabled = true;
    return btn;
}

function holidayName(id) {
    const h = (shopData.holidays || []).find(h => h.id === id);
    return h ? h.icon + ' ' + h.name : '';
}

// The button for something bought with Friendship Points
function buyButton(item, url, body) {
    if (item.owned) return ownedButton();
    if (item.locked) return lockedButton(item.unlock_at);
    const holidayLimit = item.holiday && shopData.holiday_bought;
    const btn = priceButton(item.price, '❤️', shopData.friendship_points >= item.price && !holidayLimit,
        () => shopBuy(url, body, item));
    if (holidayLimit) btn.title = 'One holiday item per visit';
    return btn;
}

function itemBadges(item) {
    return [
        item.rarity && item.rarity !== 'Common' ? [item.rarity, item.rarity.toLowerCase()] : null,
        item.season ? [seasonNames[item.season] || item.season, 'season'] : null,
        item.holiday ? [holidayName(item.holiday) || 'Holiday', 'holiday'] : null,
    ];
}

function renderEquipment(list) {
    equipmentSections.forEach(([category, title]) => {
        const items = shopData.items.filter(i => i.category === category);
        if (!items.length) return;
        list.append(el('div', 'shop-category', title));
        items.forEach(item => list.append(shopRow(item.icon, item.name, itemBadges(item),
            [[item.description], item.owned && item.lesson ? ['💡 ' + item.lesson, 'shop-lesson'] : null],
            buyButton(item, '/api/shop/buy', {item: item.id}), item.owned ? 'owned' : item.locked ? 'locked' : '')));
    });
}

function renderDecor(list) {
    [['decoration', 'Decorations (a small effect on a pet whose personality suits them)'],
        ['holiday', 'Holiday decorations']].forEach(([category, title]) => {
        const items = shopData.items.filter(i => i.category === category);
        if (!items.length) return;
        list.append(el('div', 'shop-category', title));
        items.forEach(item => list.append(shopRow(item.icon, item.name, itemBadges(item),
            [[item.description + (item.suits ? ' Suits ' + item.suits + '.' : '')],
                item.owned && item.lesson ? ['💡 ' + item.lesson, 'shop-lesson'] : null],
            buyButton(item, '/api/shop/buy', {item: item.id}), item.owned ? 'owned' : item.locked ? 'locked' : '')));
    });
}

function renderWardrobe(list) {
    list.append(el('div', 'panel-text cloud-note',
        'Cosmetics change only how your pet looks. One item per slot; what you take off is kept.'));
    slotOrder.forEach(slot => {
        const items = shopData.cosmetics.filter(c => c.slot === slot);
        if (!items.length) return;
        list.append(el('div', 'shop-category', slotNames[slot]));
        items.forEach(c => {
            let button;
            if (c.owned) {
                button = el('button', 'save-btn', c.worn ? 'Take off' : 'Wear');
                button.onclick = () => wear(slot, c.worn ? '' : c.id);
            } else {
                button = buyButton(c, '/api/cosmetics/buy', {id: c.id});
            }
            const icon = el('div', 'shop-icon cosmetic-swatch', c.icon);
            if (c.color) icon.style.setProperty('--c', c.color);
            const row = shopRow('', c.name, itemBadges(c), [c.lesson && c.owned ? ['💡 ' + c.lesson, 'shop-lesson'] : null],
                button, c.worn ? 'owned' : c.locked ? 'locked' : '');
            row.firstChild.replaceWith(icon);
            list.append(row);
        });
    });
}

function wear(slot, id) {
    postJSON('/api/wear', {slot: slot, id: id}).then(data => {
        if (!data.success) document.getElementById('shopMsg').textContent = data.error || 'Could not change the outfit';
        loadShop();
    }).catch(console.error);
}

function renderPantry(list) {
    if (!suppliesData) return;
    supplySections.forEach(([kind, title]) => {
        const supplies = suppliesData.supplies.filter(s => s.kind === kind);
        if (!supplies.length) return;
        list.append(el('div', 'shop-category', title));
        supplies.forEach(s => {
            const badges = itemBadges(s);
            if (s.discounted) badges.push(['Sale', 'sale']);
            let button;
            if (s.locked) {
                button = lockedButton(s.unlock_at);
            } else {
                const limit = s.holiday && shopData.holiday_bought;
                button = priceButton(s.price + (s.discounted ? ' (was ' + s.full_price + ')' : ''), '❤️',
                    suppliesData.points >= s.price && !limit, () => shopBuy('/api/supplies/buy', {id: s.id}, s));
            }
            const row = shopRow(s.icon, s.name, badges,
                [[s.description + (s.owned ? ' You have ' + s.owned + '.' : '')],
                    s.warning ? ['⚠️ ' + s.warning.replace(/ [^.]*\?$/, ''), 'shop-warning'] : null],
                button, s.locked ? 'locked' : '');
            if (kind === 'care' && s.owned > 0) {
                const give = el('button', 'save-btn', 'Give');
                give.onclick = () => giveCare(s.id);
                row.append(give);
            }
            list.append(row);
        });
    });
}

function giveCare(id) {
    postJSON('/api/care/give', {id: id}).then(data => {
        showPointsToast(data.success ? data.data.text : (data.error || 'Something went wrong'));
        if (shopping) loadShop(); else fetchStatus();
    }).catch(console.error);
}

function renderHouse(list) {
    list.append(el('div', 'panel-text cloud-note',
        'Home Stars are yours, not a pet\'s: earn them with trips, new places, achievements, finishing a day\'s ' +
        'goals and playing at least 10 minutes a day (' + (homeData.visited_today ? 'done today' :
            'about ' + Math.ceil(homeData.visit_seconds_left / 60) + ' more minutes today') + '; ' +
        homeData.streak + '-day streak). House upgrades help every pet.'));
    list.append(el('div', 'shop-category', 'House upgrades'));
    homeData.upgrades.forEach(u => {
        const button = u.owned ? ownedButton() :
            priceButton(u.price, '⭐', homeData.stars >= u.price, () => homeBuy({id: u.id}, u));
        list.append(shopRow(u.icon, u.name, [], [[u.description], u.owned ? ['💡 ' + u.lesson, 'shop-lesson'] : null],
            button, u.owned ? 'owned' : ''));
    });
    if (homeData.upgrades.some(u => u.id === 'air_filtration' && u.owned)) {
        const fresh = homeData.filter_days > 30;
        const button = priceButton(homeData.filter_stars, '⭐', !fresh && homeData.stars >= homeData.filter_stars,
            () => homeBuy({filter: true}));
        list.append(shopRow('🌬️', 'Air filter', [], [['The filtration\'s filter has ' +
            Math.round(homeData.filter_days) + ' game days left.' + (fresh ? '' : ' Time to replace it.')]], button));
    }
    ['wallpaper', 'floor', 'window', 'nameplate'].forEach(slot => {
        const looks = homeData.looks.filter(l => l.slot === slot);
        list.append(el('div', 'shop-category', {wallpaper: 'Wallpaper', floor: 'Floors', window: 'Window view',
            nameplate: 'Nameplate'}[slot]));
        looks.forEach(l => {
            let button;
            if (l.owned) {
                const inUse = homeData.look[slot] === l.id;
                button = el('button', 'save-btn', inUse ? 'Take down' : 'Use');
                button.onclick = () => postJSON('/api/home/look', {slot: slot, id: inUse ? '' : l.id}).then(loadShop);
            } else {
                button = priceButton(l.price, '⭐', homeData.stars >= l.price, () => homeBuy({id: l.id}));
            }
            list.append(shopRow(l.icon, l.name, [], [], button, homeData.look[slot] === l.id ? 'owned' : ''));
        });
    });
    if (homeData.recent.length) {
        list.append(el('div', 'shop-category', 'Lately'));
        homeData.recent.forEach(e => list.append(el('div', 'shop-desc',
            (e.stars > 0 ? '+' : '') + e.stars + ' ⭐ ' + e.what)));
    }
}

function shopBuy(url, body, item) {
    const msg = document.getElementById('shopMsg');
    msg.textContent = '';
    postJSON(url, body).then(data => {
        if (data.success) {
            if (item && item.lesson) showPointsToast('💡 ' + item.lesson);
        } else {
            msg.textContent = data.error || 'Could not buy this';
        }
        loadShop();
    }).catch(() => { msg.textContent = 'Connection error. Please try again.'; });
}

function homeBuy(body, item) {
    const msg = document.getElementById('shopMsg');
    msg.textContent = '';
    postJSON('/api/home/buy', body).then(data => {
        if (data.success && item && item.lesson) showPointsToast('💡 ' + item.lesson);
        if (!data.success) msg.textContent = data.error || 'Could not buy this';
        loadShop();
    }).catch(() => { msg.textContent = 'Connection error. Please try again.'; });
}

// Settings: the hemisphere seasons follow
let hemisphereAsked = false;

function askHemisphereOnce() {
    if (hemisphereAsked) return Promise.resolve();
    return fetch('/api/settings').then(r => r.json()).then(data => {
        hemisphereAsked = true;
        if (data.success && !data.data.hemisphere_set) return chooseHemisphere();
    }).catch(() => {});
}

function chooseHemisphere() {
    return new Promise(resolve => {
        showModal('🌍', 'Where do you live?',
            'Seasons in the game follow the real calendar where you live, so the shop sells the right season\'s ' +
            'items and the weather matches. Near the equator, choose either.',
            [['Northern Hemisphere', () => setHemisphere('northern').then(resolve)],
                ['Southern Hemisphere', () => setHemisphere('southern').then(resolve)]]);
    });
}

function setHemisphere(h) {
    return postJSON('/api/settings', {hemisphere: h}).then(data => {
        if (data.success) showPointsToast('Seasons follow the ' + h + ' hemisphere: it is ' +
            (seasonNames[data.data.season] || data.data.season) + '.');
    });
}

function openSettings() {
    panelMode = 'settings';
    fetch('/api/settings').then(r => r.json()).then(data => {
        if (!data.success) return;
        const p = panel();
        p.replaceChildren(panelHeader('Settings'));
        cloudText(p, 'Seasons', 'section-title');
        cloudText(p, 'The game\'s seasons, weather and seasonal breeding follow the real calendar. It is ' +
            (seasonNames[data.data.season] || data.data.season) + ' in the ' + data.data.hemisphere + ' hemisphere.',
            'panel-text');
        const row = el('div', 'cloud-row');
        ['northern', 'southern'].forEach(h => {
            const btn = cloudButton(row, (h === data.data.hemisphere ? '✓ ' : '') +
                h.charAt(0).toUpperCase() + h.slice(1) + ' Hemisphere', () => setHemisphere(h).then(openSettings));
            btn.disabled = h === data.data.hemisphere;
        });
        p.append(row);
        p.classList.remove('hidden');
    }).catch(console.error);
}

// A message in front of everything, with buttons ([label, onclick])
function showModal(icon, title, text, buttons) {
    const modal = document.getElementById('popupModal');
    document.getElementById('popupIcon').textContent = icon;
    document.getElementById('popupTitle').textContent = title;
    document.getElementById('popupText').textContent = text;
    const actions = document.getElementById('popupActions');
    actions.replaceChildren();
    buttons.forEach(([label, onclick]) => {
        const btn = el('button', 'save-btn', label);
        btn.onclick = () => { modal.classList.add('hidden'); onclick(); };
        actions.append(btn);
    });
    modal.classList.remove('hidden');
    actions.firstChild && actions.firstChild.focus();
}

// Pop-ups from the server, shown one at a time until dismissed
let popupShowing = null;

function showPopups(popups) {
    if (popupShowing || !popups || !popups.length) return;
    const p = popups[0];
    popupShowing = p.id;
    showModal(p.icon, p.title, p.text, [['OK', () => {
        postJSON('/api/popup/dismiss', {id: p.id}).finally(() => { popupShowing = null; });
    }]]);
}

// How the pet looks: what it wears drawn over it
function renderOutfit(outfit) {
    const layer = document.getElementById('petOutfit');
    if (!layer) return;
    layer.replaceChildren();
    Object.entries(outfit || {}).forEach(([slot, c]) => {
        const item = el('span', 'wear wear-' + slot + (c.color ? ' tinted' : ''));
        item.title = c.name;
        if (c.color) item.style.setProperty('--c', c.color);
        const shape = slot === 'collar' || (slot === 'neck' && c.icon === '🔻') || (slot === 'body' && c.color);
        if (shape) {
            item.classList.add('shape');
            if (slot === 'collar' && c.icon !== '⭕') item.append(el('span', 'wear-charm', c.icon));
        } else {
            item.textContent = c.icon;
        }
        layer.append(item);
    });
}

// How the home looks: wallpaper, floor, window view and nameplate
function pattern(style) {
    const [kind, a, b] = style.split(':');
    switch (kind) {
    case 'stripes': return 'repeating-linear-gradient(90deg, ' + a + ' 0 14px, ' + b + ' 14px 28px)';
    case 'dots': return 'radial-gradient(' + b + ' 2px, transparent 3px) 0 0/16px 16px, ' + a;
    case 'boards': return 'repeating-linear-gradient(90deg, ' + a + ' 0 38px, ' + b + ' 38px 40px)';
    case 'checks': return 'repeating-conic-gradient(' + a + ' 0 25%, ' + b + ' 0 50%) 0 0/24px 24px';
    }
    return style;
}

function renderHouseLook(look, name) {
    const scene = document.getElementById('petScene');
    if (!scene) return;
    look = look || {};
    scene.style.setProperty('--wall', look.wallpaper ? pattern(look.wallpaper.style) : 'transparent');
    scene.style.setProperty('--floor', look.floor ? pattern(look.floor.style) : 'transparent');
    scene.classList.toggle('decorated', Object.keys(look).length > 0);
    document.getElementById('sceneWindow').textContent = look.window ? look.window.style : '';
    const plate = document.getElementById('sceneNameplate');
    plate.textContent = look.nameplate ? name : '';
    plate.style.background = look.nameplate ? look.nameplate.style : '';
}

// Body panel extras: upkeep and the first-aid kit
function renderUpkeep(status) {
    const box = document.getElementById('upkeepLine');
    if (!box) return;
    box.replaceChildren();
    (status.upkeep || []).forEach(u => {
        const due = u.days_left <= 0;
        box.append(el('div', 'upkeep' + (due ? ' due' : ''), (due ? '⚠️ ' : '✓ ') + u.name + ': ' +
            (due ? 'due now (in the pantry shop)' : Math.round(u.days_left) + ' game days left')));
    });
    const aid = document.getElementById('firstAidBtn');
    aid.classList.toggle('hidden', !status.is_alive || !status.injured || !status.first_aid_uses);
    aid.textContent = '🩹 Use first-aid kit (' + status.first_aid_uses + ' left)';
}

// Family: giving points to young children, and passing belongings on
function openFamily() {
    panelMode = 'family';
    fetch('/api/family').then(r => r.json()).then(data => {
        if (data.success) renderFamily(data.data);
    }).catch(console.error);
}

function minutesText(seconds) {
    const m = Math.round(seconds / 60);
    return m < 1 ? 'less than a minute' : m === 1 ? 'about a minute' : 'about ' + m + ' minutes';
}

function renderFamily(f) {
    const p = panel();
    p.replaceChildren(panelHeader('Family'));
    if (f.young && f.parents.length) {
        cloudText(p, f.name + ' is young for ' + minutesText(f.young_seconds_left) + ' more of play while it is the ' +
            'active pet. Until then its parents can give it Friendship Points from their own Family panel.', 'panel-text');
        f.parents.forEach(parent => cloudText(p, '👪 ' + parent.name + ' has ' + parent.points + ' points.', 'panel-text'));
    }
    if (!f.children.length) {
        cloudText(p, f.name + ' has no children' + (f.young ? '.' : ' yet.'), 'panel-text cloud-note');
    }
    if (!f.alive && f.passed_on_to) {
        cloudText(p, f.name + '\'s belongings now belong to ' + f.passed_on_to + '.', 'panel-text');
    } else if (!f.alive && f.can_pass_on) {
        cloudText(p, 'Pass ' + f.name + '\'s belongings (equipment, decorations, cosmetics and unopened supplies) ' +
            'and Friendship Points on to one of its children:', 'panel-text');
    } else if (f.alive && f.children.length) {
        cloudText(p, 'Give Friendship Points to a young child. A grown-up child can be given points only after ' +
            f.name + ' dies. ' + f.name + ' has ' + f.points + ' points.', 'panel-text cloud-note');
    }
    f.children.forEach(child => {
        const row = el('div', 'family-row');
        row.append(el('span', 'family-name', speciesIcon(child.species) + child.name + (child.alive ? '' : ' (passed away)') +
            (child.young ? ' · young for ' + minutesText(child.young_seconds_left) + ' of its play' : ' · grown up')));
        if (child.can_transfer && f.points > 0 && (f.alive || f.can_pass_on)) {
            const amount = el('input', 'family-amount');
            amount.type = 'number';
            amount.min = 0;
            amount.max = f.points;
            amount.value = f.alive ? Math.min(f.points, 50) : f.points;
            amount.setAttribute('aria-label', 'Points to give ' + child.name);
            row.append(amount);
            if (f.alive) {
                cloudButton(row, 'Give', () => familyPost('/api/family/transfer',
                    {child_slot: child.slot, amount: Number(amount.value)}));
            } else {
                cloudButton(row, 'Pass on', () => familyPost('/api/family/pass_on',
                    {child_slot: child.slot, points: Number(amount.value)}));
            }
        } else if (!f.alive && f.can_pass_on && child.alive) {
            cloudButton(row, 'Pass on', () => familyPost('/api/family/pass_on', {child_slot: child.slot, points: 0}));
        }
        p.append(row);
    });
    p.classList.remove('hidden');
}

function familyPost(url, body) {
    postJSON(url, body).then(data => {
        showPointsToast(data.success ? data.data.text : (data.error || 'Something went wrong'));
        openFamily();
        fetchStatus();
    }).catch(console.error);
}
