// Browser smoke test: plays the game's page in a headless browser and
// fails on any error in the page. It creates a cat and a dog through the
// form, gives every kind of care, opens every panel from its button,
// plays every game and answers a home happening, then adds a second
// player, plays as them and switches back.
//
// Usage: node smoke.js <game address>, with Playwright where node finds it
// (NODE_PATH). The Go test TestBrowserSmoke starts a game and runs it:
//   GOCHI_E2E=1 go test ./cmd/gochiserver -run TestBrowserSmoke

const { chromium } = require('playwright');

const base = process.argv[2] || 'http://127.0.0.1:8080';
const problems = [];
// Set while the page reloads after a player switch: requests still running
// then are cut off, which the page logs, and which isn't a problem
let reloading = false;

async function main() {
    const launch = {};
    if (process.env.PLAYWRIGHT_CHROMIUM) launch.executablePath = process.env.PLAYWRIGHT_CHROMIUM;
    const browser = await chromium.launch(launch);
    const page = await browser.newPage({viewport: {width: 420, height: 900}});
    page.on('pageerror', e => problems.push('page error: ' + e.message));
    page.on('console', m => {
        // The game answers refused actions with 4xx, which the browser logs
        if (reloading && /Failed to fetch/.test(m.text())) return;
        if (m.type() === 'error' && !/status of 4\d\d/.test(m.text())) problems.push('console: ' + m.text());
    });
    page.on('dialog', d => d.dismiss());
    page.on('response', r => {
        // Refusals from the game's routes are expected; anything else failing to load is not
        if (r.status() >= 400 && !new URL(r.url()).pathname.startsWith('/api/')) {
            problems.push('could not load ' + r.url() + ': ' + r.status());
        }
    });

    await page.goto(base + '/');
    await page.waitForTimeout(1000);
    for (const [name, species] of [['Biscuit', 'dog'], ['Mochi', 'cat']]) {
        if (await page.locator('#petView').isVisible()) await page.evaluate(() => openChooser());
        await page.fill('#petNameInput', name);
        await page.evaluate(s => {
            const radio = document.querySelector('input[name="species"][value="' + s + '"]');
            if (radio) { radio.checked = true; radio.dispatchEvent(new Event('change', {bubbles: true})); }
        }, species);
        await page.click('#createBtn');
        await page.waitForTimeout(1500);
        await settle(page);
        await playEverything(page, name);
    }
    await groomPepper(page);
    await secondPlayer(page);
    await browser.close();
}

// secondPlayer adds Sam, who starts with no pets, gives them a pet, switches
// back to Player 1, whose pets are still there, and deletes Sam
async function secondPlayer(page) {
    const playerRow = name => page.locator('#playersPanel .player-row', {hasText: name});
    await page.click('#playersBtn');
    await page.fill('#newPlayerName', 'Sam');
    await page.locator('#playersPanel button', {hasText: 'Add player'}).click();
    await page.waitForTimeout(800);
    reloading = true;
    const reloadedSam = page.waitForEvent('load');
    await playerRow('Sam').locator('button', {hasText: 'Play'}).click();
    await reloadedSam;
    await page.waitForTimeout(1500);
    reloading = false;
    if (!/^👤 Sam #\d{4}$/.test(await page.textContent('#playersBtn'))) {
        problems.push('the Players button should name Sam: ' + await page.textContent('#playersBtn'));
    }
    if (!await page.locator('#createView').isVisible() || await page.locator('#savedPetList li').count() !== 0) {
        problems.push("Sam shouldn't have Player 1's pets");
    }
    await page.fill('#petNameInput', 'Rex');
    await page.click('#createBtn');
    await page.waitForTimeout(1500);
    await settle(page);

    await page.click('#playersBtn');
    await page.waitForTimeout(500);
    reloading = true;
    const reloadedPlayer1 = page.waitForEvent('load');
    await playerRow('Player 1').locator('button', {hasText: 'Play'}).click();
    await reloadedPlayer1;
    await page.waitForTimeout(1500);
    reloading = false;
    await page.evaluate(() => openChooser());
    await page.waitForTimeout(800);
    const names = await page.locator('#savedPetList li').allTextContents();
    if (!names.some(n => n.includes('Pepper')) || names.some(n => n.includes('Rex'))) {
        problems.push("Player 1's pets should be back, without Sam's: " + names.join(' | '));
    }
    await page.evaluate(() => closeChooser());

    await page.click('#playersBtn');
    await page.waitForTimeout(500);
    await playerRow('Sam').locator('button', {hasText: 'Delete'}).click();
    await playerRow('Sam').locator('input').fill('Sam');
    await playerRow('Sam').locator('button', {hasText: 'Delete for good'}).click();
    await page.waitForTimeout(800);
    if (await playerRow('Sam').count() !== 0) problems.push('Sam should be deleted');
    await page.evaluate(() => closePlayers());
    console.log('Added Sam, played as them, switched back and deleted them');
}

// groomPepper loads Pepper, a cat the browser test saved groomed before and
// in need of it again, and plays the grooming game with her (a new pet is
// clean, and one grooming makes a pet clean, so it would turn the game down)
async function groomPepper(page) {
    await page.evaluate(() => openChooser());
    const load = page.locator('#savedPetList li', {hasText: 'Pepper'}).locator('button', {hasText: 'Load'});
    if (await load.count() === 0) {
        problems.push('Pepper is not in the list of saved pets');
        return;
    }
    await load.click();
    await page.waitForTimeout(1500);
    await settle(page);
    const games = await page.evaluate(() => fetch('/api/games').then(r => r.json()).then(d => d.data || []));
    const grooming = games.find(g => g.id === 'grooming');
    if (!grooming || grooming.unavailable) {
        problems.push('Pepper can not play the grooming game: ' + (grooming ? grooming.unavailable : 'no such game'));
        return;
    }
    await playGame(page, grooming);
    console.log('Played grooming with Pepper');
    await dressPepper(page);
    await drawEveryCosmetic(page);
}

// dressPepper dresses Pepper, who owns every cosmetic, through the
// wardrobe: an item in each slot, which must then all show on her
async function dressPepper(page) {
    const cosmetics = await page.evaluate(() => fetch('/api/shop').then(r => r.json()).then(d => d.data.cosmetics));
    const unowned = cosmetics.filter(c => !c.owned).map(c => c.id);
    if (unowned.length) problems.push('Pepper should own every cosmetic, not: ' + unowned.join(', '));
    const outfit = {};
    cosmetics.forEach(c => { if (!outfit[c.slot]) outfit[c.slot] = c; });
    if (Object.keys(outfit).length !== 6) problems.push('the wardrobe should have six slots: ' + Object.keys(outfit));

    await page.evaluate(() => openShop());
    await page.waitForTimeout(800);
    await settle(page);
    await page.locator('.shop-tab', {hasText: 'Wardrobe'}).click();
    for (const c of Object.values(outfit)) {
        const row = page.locator('#shopList .shop-item').filter({has: page.locator('.shop-name', {hasText: c.name})}).first();
        await row.locator('button', {hasText: 'Wear'}).click({timeout: 5000})
            .catch(e => problems.push('could not wear ' + c.name + ': ' + e.message.split('\n')[0]));
        await page.waitForTimeout(500); // The wardrobe reloads
    }
    const worn = await page.locator('#shopList .shop-item button', {hasText: 'Take off'}).count();
    if (worn !== 6) problems.push('the wardrobe should show six items to take off, not ' + worn);
    await page.evaluate(() => closeShop());
    await page.waitForTimeout(1500); // The pet is drawn again with the next status
    const shown = await page.evaluate(() => [...document.querySelectorAll('#petSprite title')].map(t => t.textContent));
    for (const c of Object.values(outfit)) {
        if (!shown.includes(c.name)) problems.push('Pepper should be shown wearing ' + c.name);
    }
    console.log('Dressed Pepper in ' + Object.values(outfit).map(c => c.name).join(', '));
}

// drawEveryCosmetic draws every cosmetic on every breed of both kinds of
// pet, with the page's own drawing, and checks each drawing is well formed
// and shows the item
async function drawEveryCosmetic(page) {
    const result = await page.evaluate(async () => {
        const cosmetics = (await fetch('/api/shop').then(r => r.json())).data.cosmetics;
        const found = [];
        let draws = 0;
        for (const species of ['cat', 'dog']) {
            const breeds = (await fetch('/api/breeds?species=' + species).then(r => r.json())).data.breeds;
            for (const b of breeds) {
                for (const c of cosmetics) {
                    const svg = GochiArt.draw({species: species, look: b.typical, outfit: {[c.slot]: c}, size: 100, label: b.name});
                    draws++;
                    const doc = new DOMParser().parseFromString(svg, 'image/svg+xml');
                    if (doc.querySelector('parsererror')) found.push(c.id + ' on a ' + b.name + ' is not well formed');
                    else if (![...doc.querySelectorAll('title')].some(t => t.textContent === c.name)) found.push(c.id + ' is not drawn on a ' + b.name);
                }
            }
        }
        return {draws: draws, cosmetics: cosmetics.length, found: found};
    });
    // Round the neck, in layers: with a collar that has a charm, a tag and
    // each kind of neck item (a bandana, or one drawn as its icon), the
    // charms hang from the collar, on top of the neck item, and don't
    // overlap each other
    const overlaps = await page.evaluate(async () => {
        const cosmetics = (await fetch('/api/shop').then(r => r.json())).data.cosmetics;
        const byID = id => cosmetics.find(c => c.id === id);
        const necks = ['bandana_blue', 'bow_tie', 'knitted_scarf'].map(byID);
        const box = document.createElement('div');
        document.body.append(box);
        const found = [];
        for (const species of ['cat', 'dog']) {
            const breeds = (await fetch('/api/breeds?species=' + species).then(r => r.json())).data.breeds;
            for (const b of breeds) {
                for (const neck of necks) {
                    const outfit = {collar: byID('collar_bell'), tag: byID('tag_paw'), neck: neck};
                    box.innerHTML = GochiArt.draw({species: species, look: b.typical, outfit: outfit, size: 200});
                    const shape = (name, tag) => [...box.querySelectorAll('title')]
                        .find(t => t.textContent === name && (!tag || t.parentElement.tagName === tag)).parentElement;
                    const bell = shape(outfit.collar.name, 'text'), paw = shape(outfit.tag.name), scarf = shape(neck.name);
                    const band = shape(outfit.collar.name, 'path').getBBox();
                    const on = b.name + ' with a ' + neck.name;
                    const [bb, pb] = [bell.getBBox(), paw.getBBox()];
                    if (bb.x < pb.x + pb.width - 1 && pb.x < bb.x + bb.width - 1) found.push('the bell and the tag overlap on a ' + on);
                    for (const [charm, r] of [[bell, bb], [paw, pb]]) {
                        if (!(scarf.compareDocumentPosition(charm) & Node.DOCUMENT_POSITION_FOLLOWING)) found.push('the ' + neck.name + ' is drawn over a charm on a ' + on);
                        if (Math.abs(r.y - (band.y + band.height)) > 3) found.push('a charm doesn\'t hang from the collar on a ' + on);
                    }
                }
            }
        }
        box.remove();
        return found;
    });
    result.found.push(...overlaps);
    result.found.slice(0, 20).forEach(p => problems.push(p));
    if (result.found.length > 20) problems.push((result.found.length - 20) + ' more drawing problems');
    console.log('Drew ' + result.cosmetics + ' cosmetics on every breed: ' + result.draws + ' drawings');
}

// settle closes the tour and any pop-ups, the way a player would
async function settle(page) {
    for (let i = 0; i < 12; i++) {
        if (await page.locator('#tourBox').isVisible().catch(() => false)) {
            await page.locator('#tourBox button', {hasText: 'Skip'}).click();
        } else if (await page.locator('#popupModal').isVisible()) {
            await page.locator('#popupActions button').last().click();
        } else {
            return;
        }
        await page.waitForTimeout(250);
    }
}

// opensInView clicks each visible, enabled button matching selector, with
// the page scrolled so the button is at the bottom of the window, and
// checks that the panel or view it opens, if any, has its top in sight
async function opensInView(page, selector) {
    const places = ['#activityPanel', '#playersPanel', '#shopView', '#createView'];
    const count = await page.locator(selector).count();
    for (let i = 0; i < count; i++) {
        const button = page.locator(selector).nth(i);
        await closePanels(page);
        await page.evaluate(() => { closePlayers(); if (choosing) closeChooser(); });
        await settle(page);
        if (!(await button.isVisible()) || !(await button.isEnabled())) continue;
        const label = (await button.innerText()).trim();
        const shownBefore = await page.evaluate(ids => ids.filter(id => !document.querySelector(id).classList.contains('hidden')), places);
        await button.evaluate(b => b.scrollIntoView({block: 'end'}));
        await button.click({timeout: 5000}).catch(() => {});
        await page.waitForTimeout(1500); // It loads, then the page scrolls
        await settle(page);
        const opened = await page.evaluate(([ids, before]) => ids.filter(id => {
            const el = document.querySelector(id);
            return !el.classList.contains('hidden') && (!before.includes(id) || id === '#activityPanel' || id === '#playersPanel');
        }).map(id => {
            const r = document.querySelector(id).getBoundingClientRect();
            return {id, inView: r.top > -1 && r.top < innerHeight};
        }), [places, shownBefore]);
        opened.filter(o => !o.inView).forEach(o => problems.push(label + ' opened ' + o.id + ' out of view'));
        if (await page.locator('#shopView').isVisible()) await page.evaluate(() => closeShop());
    }
    await closePanels(page);
    await page.evaluate(() => { closePlayers(); if (choosing) closeChooser(); });
}

// clickAll clicks each visible, enabled button matching selector
async function clickAll(page, selector, between) {
    const count = await page.locator(selector).count();
    for (let i = 0; i < count; i++) {
        const button = page.locator(selector).nth(i);
        await settle(page);
        if (!(await button.isVisible()) || !(await button.isEnabled())) continue;
        const label = (await button.innerText()).trim();
        await button.click({timeout: 5000})
            .catch(e => problems.push('could not click ' + label + ': ' + e.message.split('\n')[0]));
        await page.waitForTimeout(600); // First-time tips appear after 400ms
        await settle(page);
        if (between) await between(label);
    }
}

// playGame plays a game from the page, as a player would, and checks it
// finishes
async function playGame(page, game) {
    const playing = () => page.evaluate(() => panelMode === 'game');
    await page.evaluate(g => startGame(g), game);
    await page.waitForTimeout(300);
    const panel = page.locator('#activityPanel');
    if (game.id === 'fetch') {
        for (let i = 0; i < 5; i++) {
            await panel.locator('button', {hasText: 'Throw'}).click({timeout: 5000});
            await page.waitForTimeout(150);
        }
    } else if (game.id === 'wand') {
        await panel.locator('button', {hasText: 'Let it catch'}).click();
    } else if (game.id === 'puzzle' || game.id === 'scent') {
        for (let i = 0; i < 40 && await playing(); i++) {
            const tiles = panel.locator('.game-tile:enabled');
            if (await tiles.count() === 0) break;
            const clicked = await tiles.nth(i % await tiles.count()).click({timeout: 3000}).then(() => true, () => false);
            if (!clicked && await playing()) { // Not just the game ending as the tile was clicked
                problems.push('could not click a tile in ' + game.id);
                break;
            }
            await page.waitForTimeout(60);
        }
    } else if (game.id === 'grooming') {
        const rows = panel.locator('.trick-row');
        for (let i = 0; i < await rows.count(); i++) await rows.nth(i).locator('button').first().click();
    }
    // A game shows how it went a moment after it ends
    for (let wait = 0; wait < 20 && await playing(); wait++) await page.waitForTimeout(150);
    if (await playing()) problems.push(game.id + ' did not finish');
    await page.waitForTimeout(1300);
    await closePanels(page);
}

// chooseFirst opens the choices behind a button and makes the first one,
// as a player would
async function chooseFirst(page, opener, what) {
    await settle(page);
    await page.locator(opener).click();
    const choice = page.locator('#activityPanel .choice-row button').first();
    await choice.click({timeout: 5000})
        .catch(e => problems.push('could not choose ' + what + ': ' + e.message.split('\n')[0]));
    await page.waitForTimeout(600); // First-time tips appear after 400ms
    await settle(page);
    await closePanels(page);
}

async function closePanels(page) {
    await page.evaluate(() => { closePanel(); if (typeof closeShop === 'function') closeShop(); });
}

async function playEverything(page, name) {
    // Care
    await clickAll(page, '.actions .action-btn');
    // Buttons with choices (▾) show them in a panel above the buttons: it must
    // come into view even with the buttons at the top of the window
    for (const kind of ['feed', 'pet', 'play']) {
        await page.$eval('.action-btn.' + kind, b => { b.scrollIntoView({block: 'start'}); b.click(); });
        await page.waitForTimeout(1200); // The choices load, then the page scrolls
        const seen = await page.$eval('#activityPanel', p => {
            const r = p.getBoundingClientRect();
            return !p.classList.contains('hidden') && r.top > -1 && r.bottom < innerHeight + 1;
        });
        if (!seen) problems.push('the ' + kind + ' choices opened out of view');
        await closePanels(page);
    }
    // Things to do now: listed under the care buttons, each ready one
    // opening its activity (not exploring, which sends the pet away)
    if (!(await page.locator('#thingsToDo').isVisible()) || await page.locator('#thingsToDo .thing-btn').count() === 0) {
        problems.push(name + ' has nothing listed to do now');
    }
    await clickAll(page, '#thingsToDo .thing-btn:not([data-thing="explore"])', async () => closePanels(page));
    // The care that opens the games: play (from Play's choices) and
    // enrichment (from Enrich's); grooming was given above
    await chooseFirst(page, '.action-btn.play', 'play');
    await chooseFirst(page, '.activity-btn.enrich', 'enrichment');
    // Whatever a button opens comes into view, from a page scrolled to the button
    await opensInView(page, '#thingsToDo .thing-btn:not([data-thing="explore"])');
    await opensInView(page, '.activities .activity-btn:not(.ar)');
    await opensInView(page, '.save-row button:not([onclick*="AR"])');
    await opensInView(page, '#playersBtn');
    // Every activity, and every button along the bottom, but not the camera or ones that leave the pet
    await clickAll(page, '.activities .activity-btn:not(.ar)', async () => closePanels(page));
    await clickAll(page, '.save-row button:not([onclick*="Chooser"]):not([onclick*="AR"])', async label => {
        if (/Shop/.test(label)) {
            await clickAll(page, '.shop-tab');
        }
        await closePanels(page);
    });
    // Every game the pet can play now (call it home first: panels close while it is away)
    await page.evaluate(() => postJSON('/api/explore/recall', {}));
    await page.waitForTimeout(1500);
    const games = await page.evaluate(() => fetch('/api/games').then(r => r.json()).then(d => d.data || []));
    const playable = games.filter(g => !g.unavailable);
    console.log(name + ' can play ' + playable.map(g => g.id).join(', '));
    games.filter(g => g.unavailable).forEach(g => console.log('  ' + g.id + ': ' + g.unavailable));
    if (playable.length === 0) problems.push(name + ' has no game to play');
    for (const game of playable) await playGame(page, game);
    // A home happening, from the page
    await page.evaluate(() => showHappening({id: 'thunderstorm', icon: '⛈️', title: 'Thunderstorm', text: 'A test',
        choices: [{id: 'comfort', label: 'Comfort it'}]}));
    await page.waitForTimeout(300);
    await page.locator('#popupActions button').first().click();
    await page.waitForTimeout(800);
    await settle(page);
    // The pet list, the book and the help again
    await page.evaluate(() => { openChooser(); });
    await page.waitForTimeout(800);
    await page.evaluate(() => { closeChooser(); openBook(); });
    await page.waitForTimeout(800);
    await closePanels(page);
    console.log('Played ' + name);
}

main().then(() => {
    if (problems.length) {
        console.log(JSON.stringify(problems, null, 2));
        process.exit(1);
    }
    console.log('No problems');
}).catch(e => {
    console.log('The smoke test failed to run: ' + e.message);
    process.exit(2);
});
