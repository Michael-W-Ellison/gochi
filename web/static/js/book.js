// Keepsake books, the theme of the week and long-term projects. Books stay
// on this computer. Uses panel helpers from the page, el() from shop.js
// and GochiArt from pet-art.js.

// bookArt draws a pet as it looked on a page
function bookArt(species, look, wearing, size, state) {
    const holder = el('span', 'book-art');
    holder.innerHTML = GochiArt.draw({species: species, look: look, outfit: wearing || {}, size: size || 64,
        state: state || 'idle', label: ''});
    return holder;
}

function openBooks() {
    panelMode = 'books';
    fetch('/api/books').then(r => r.json()).then(data => {
        if (!data.success) return;
        const p = panel();
        p.replaceChildren(panelHeader('📔 Books'));
        cloudText(p, 'Each pet\'s book of its life. Books are kept on this computer only, and stay after a pet is ' +
            'deleted until you delete them too.', 'panel-text cloud-note');
        if (!data.data.length) cloudText(p, 'No books yet.', 'panel-text');
        data.data.forEach(b => {
            const row = el('div', 'trick-row');
            row.append(bookArt(b.species, b.look, null, 40, b.alive ? 'idle' : 'dead'),
                el('span', 'trick-name', b.name + (b.alive ? '' : ' 🕊️')), el('span', 'cloud-note', b.pages + ' pages'));
            cloudButton(row, 'Open', () => openBook(b.id));
            if (!b.saved) {
                cloudButton(row, 'Delete', () => showModal('📔', 'Delete ' + b.name + '\'s book?',
                    'This deletes the book for good. ' + b.name + ' is no longer saved, so it can\'t be made again.',
                    [['Delete', () => postJSON('/api/books/delete', {id: b.id}).then(openBooks)], ['Keep it', () => {}]]));
            }
            p.append(row);
        });
        p.classList.remove('hidden');
    }).catch(console.error);
}

function openBook(id) {
    id = id || (lastStatus && lastStatus.id);
    if (!id) return;
    panelMode = 'book';
    fetch('/api/books?id=' + encodeURIComponent(id)).then(r => r.json()).then(data => {
        if (!data.success) {
            showPointsToast(data.error || 'This book isn\'t written yet; it is saved with the pet.');
            return;
        }
        playSound('page');
        renderBook(data.data);
    }).catch(console.error);
}

function renderBook(b) {
    const p = panel();
    p.replaceChildren(panelHeader('📔 ' + b.name + '\'s book'));
    const cover = el('div', 'book-cover');
    cover.append(bookArt(b.species, b.look, null, 96, b.alive ? 'idle' : 'dead'));
    p.append(cover);

    // The family tree
    const family = [['Grandparents', b.grandparents], ['Parents', b.parents], ['Children', b.children]]
        .filter(([, list]) => list && list.length);
    if (family.length) {
        cloudText(p, 'Family', 'section-title');
        family.forEach(([title, list]) => {
            const row = el('div', 'book-family');
            row.append(el('span', 'cloud-note', title + ': '));
            list.forEach(f => {
                const who = el('button', 'book-relative');
                who.append(bookArt(f.species, f.look, null, 36, f.alive ? 'idle' : 'dead'), ' ' + f.name);
                who.onclick = () => openBook(f.id);
                row.append(who);
            });
            p.append(row);
        });
    }

    // The pages
    cloudText(p, 'Pages', 'section-title');
    (b.pages || []).forEach(page => {
        const row = el('div', 'book-page');
        row.append(bookArt(b.species, page.look || b.look, page.wearing, 56));
        const words = el('div', '');
        words.append(el('div', 'panel-text', page.text),
            el('div', 'cloud-note', Math.floor(page.age) + ' game days old · ' +
                new Date(page.day + 'T12:00:00').toLocaleDateString()));
        row.append(words);
        p.append(row);
    });

    // A gentle closing page
    if (b.memorial) {
        const m = b.memorial;
        const page = el('div', 'book-memorial');
        page.append(el('div', 'show-title', (b.candle ? '🕯️ ' : '') + 'In loving memory of ' + b.name));
        const lines = ['Lived ' + Math.floor(m.age) + ' game days.',
            m.tricks ? 'Learned ' + m.tricks + (m.tricks === 1 ? ' trick.' : ' tricks.') : '',
            m.places ? 'Explored ' + m.places + (m.places === 1 ? ' place.' : ' places.') : '',
            m.ribbons ? 'Won ' + m.ribbons + (m.ribbons === 1 ? ' ribbon.' : ' ribbons.') : '',
            m.favorite ? 'Loved ' + m.favorite.toLowerCase() + ' most.' : '',
            m.best_friend ? 'Closest to ' + m.best_friend + '.' : ''].filter(Boolean);
        page.append(el('div', 'panel-text', lines.join(' ')));
        const row = el('div', '');
        cloudButton(row, b.candle ? 'Let the candle go out' : '🕯️ Light a candle', () =>
            postJSON('/api/books/candle', {id: b.id, lit: !b.candle}).then(() => openBook(b.id)));
        page.append(row);
        p.append(page);
    }
    const row = el('div', 'save-row');
    cloudButton(row, 'All books', openBooks);
    p.append(row);
    p.classList.remove('hidden');
}

// The theme of the week, under the daily goals
function renderTheme(status) {
    const box = document.getElementById('themeLine');
    if (!box || !status.theme || !status.theme.id) return;
    const t = status.theme;
    box.replaceChildren(el('strong', '', t.icon + ' ' + t.name + ': '), t.text + ' ');
    const read = el('button', 'link-btn', 'Read about it');
    read.onclick = () => openGuide(t.topic);
    box.append(read);
}

function openProjects() {
    panelMode = 'projects';
    fetch('/api/projects').then(r => r.json()).then(data => {
        if (!data.success) return;
        const p = panel();
        p.replaceChildren(panelHeader('🎯 Projects'));
        cloudText(p, 'Long-term goals for the whole household.', 'panel-text cloud-note');
        data.data.forEach(pr => {
            const box = el('div', 'show-box');
            box.append(el('div', 'show-title', pr.icon + ' ' + pr.name + (pr.progress >= pr.target ? ' ✅' : '')));
            const bar = el('div', 'trick-bar');
            const fill = el('div', 'trick-fill');
            fill.style.width = Math.round(100 * Math.min(pr.progress / Math.max(pr.target, 1), 1)) + '%';
            bar.append(fill);
            box.append(bar, el('div', 'cloud-note', pr.detail + ' ' + pr.progress + ' of ' + pr.target + '.'));
            p.append(box);
        });
        p.classList.remove('hidden');
    }).catch(console.error);
}
