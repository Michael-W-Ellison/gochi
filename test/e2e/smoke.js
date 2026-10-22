// Browser smoke test: plays the game's page in a headless browser and
// fails on any error in the page. It creates a cat and a dog through the
// form, gives every kind of care, opens every panel from its button,
// plays every game and answers a home happening.
//
// Usage: node smoke.js <game address>, with Playwright where node finds it
// (NODE_PATH). The Go test TestBrowserSmoke starts a game and runs it:
//   GOCHI_E2E=1 go test ./cmd/gochiserver -run TestBrowserSmoke

const { chromium } = require('playwright');

const base = process.argv[2] || 'http://127.0.0.1:8080';
const problems = [];

async function main() {
    const launch = {};
    if (process.env.PLAYWRIGHT_CHROMIUM) launch.executablePath = process.env.PLAYWRIGHT_CHROMIUM;
    const browser = await chromium.launch(launch);
    const page = await browser.newPage({viewport: {width: 420, height: 900}});
    page.on('pageerror', e => problems.push('page error: ' + e.message));
    page.on('console', m => {
        // The game answers refused actions with 4xx, which the browser logs
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
    await browser.close();
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
    for (const game of playable) {
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
            for (let i = 0; i < 40 && await page.evaluate(() => panelMode) === 'game'; i++) {
                const tiles = panel.locator('.game-tile:enabled');
                if (await tiles.count() === 0) break;
                await tiles.nth(i % await tiles.count()).click();
                await page.waitForTimeout(60);
            }
        } else if (game.id === 'grooming') {
            const rows = panel.locator('.trick-row');
            for (let i = 0; i < await rows.count(); i++) await rows.nth(i).locator('button').first().click();
        }
        await page.waitForTimeout(1300);
        await closePanels(page);
    }
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
