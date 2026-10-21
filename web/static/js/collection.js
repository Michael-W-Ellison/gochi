// The household's souvenir collection, the shelf, and gifts between pets.
// Uses panel helpers from the page and el() from shop.js.

const setPlaceNames = {park: 'Park', beach: 'Beach', farm: 'Farm', forest: 'Forest', city: 'City', mountain: 'Mountain',
    meadow: 'Meadow', river: 'River', dog_park: 'Dog park', garden: 'Garden', snowy_hill: 'Snowy hill', lakeside: 'Lakeside'};

function openCollection() {
    panelMode = 'collection';
    Promise.all([fetch('/api/collection').then(r => r.json()), fetch('/api/saves').then(r => r.json())])
        .then(([collection, saves]) => {
            if (collection.success) {
                const pets = (saves.success && saves.data.pets) || [];
                renderCollection(collection.data, pets.filter(pet => pet.slot !== saves.data.active_slot));
            }
        }).catch(console.error);
}

// The shelf in the pet's scene
function renderShelf(shelf) {
    const box = document.getElementById('sceneShelf');
    if (!box) return;
    box.textContent = (shelf || []).map(s => s.icon).join('');
    box.title = (shelf || []).map(s => s.name).join(', ');
    box.classList.toggle('hidden', !(shelf || []).length);
}

function renderCollection(data, pets) {
    const p = panel();
    p.replaceChildren(panelHeader('🧺 Collection'));
    cloudText(p, 'Everything your pets have brought home. Finding a place\'s four souvenirs completes its set ' +
        '(+10 ⭐), and each rare one earns +10 ⭐.', 'panel-text cloud-note');
    const byId = {};
    data.places.forEach(set => {
        set.souvenirs.forEach(s => { byId[s.id] = s; });
        const box = el('div', 'collection-set' + (set.complete ? ' complete' : ''));
        box.append(el('div', '', (setPlaceNames[set.place] || set.place) + (set.complete ? ' ✅' : '') +
            ' · ' + set.found.length + '/' + set.souvenirs.length));
        const items = el('div', 'collection-items');
        set.souvenirs.forEach(s => {
            const found = set.found.includes(s.id);
            const item = el('span', found ? '' : 'missing', s.icon);
            item.title = found ? s.name + (s.rare ? ' (rare)' : '') : '?';
            items.append(item);
        });
        box.append(items);
        p.append(box);
    });

    // The shelf: up to five found souvenirs
    const found = data.places.flatMap(set => set.found);
    p.append(el('div', 'section-title', 'Shelf'));
    cloudText(p, 'Choose up to 5 souvenirs to show. A whole place\'s set cheers your pets up a little.', 'panel-text cloud-note');
    const picks = el('div', '');
    const boxes = [];
    found.forEach(id => {
        const label = el('label', 'shelf-pick');
        const box = el('input');
        box.type = 'checkbox';
        box.value = id;
        box.checked = data.shelf.includes(id);
        box.setAttribute('aria-label', byId[id].name);
        boxes.push(box);
        label.append(box, byId[id].icon);
        label.title = byId[id].name;
        picks.append(label);
    });
    if (!found.length) picks.append(el('span', 'cloud-note', 'Nothing found yet: go exploring!'));
    p.append(picks);
    if (found.length) {
        cloudButton(p, 'Put on the shelf', () => {
            const items = boxes.filter(b => b.checked).map(b => b.value);
            postJSON('/api/home/shelf', {items}).then(r => {
                showPointsToast(r.success ? (r.data.set ? 'A whole set on the shelf! 🧺' : 'The shelf is arranged.') : r.error);
                if (r.success) fetchStatus();
            });
        });
    }

    // Gifts
    const others = pets.filter(pet => pet.is_alive && !pet.unreadable);
    if (data.mine.length && others.length) {
        p.append(el('div', 'section-title', 'Give a gift'));
        const row = el('div', 'show-row');
        const what = el('select', '');
        what.setAttribute('aria-label', 'Souvenir');
        [...new Set(data.mine)].forEach(id => {
            const o = el('option', '', byId[id] ? byId[id].icon + ' ' + byId[id].name : id);
            o.value = id;
            what.append(o);
        });
        const who = el('select', '');
        who.setAttribute('aria-label', 'To');
        others.forEach(pet => {
            const o = el('option', '', pet.name);
            o.value = pet.slot;
            who.append(o);
        });
        row.append(what, el('span', '', 'to'), who);
        cloudButton(row, 'Give', () => postJSON('/api/souvenirs/gift', {souvenir: what.value, to_slot: who.value}).then(r => {
            showPointsToast(r.success ? r.data.text : r.error);
            if (r.success) openCollection();
        }));
        p.append(row);
    }
    p.classList.remove('hidden');
}
