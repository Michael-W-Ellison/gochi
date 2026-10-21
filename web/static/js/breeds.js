// Breeds: choosing one for a new pet, and what the About panel says about
// a pet's breed. Uses el() from shop.js and GochiArt from pet-art.js.

let breedData = null;      // From /api/breeds for the chosen species
let chosenBreed = 'random'; // A breed ID, or "random" for any unlocked one

// Loads the breeds of a species and shows them in the new pet form
function loadBreeds(species) {
    return fetch('/api/breeds?species=' + encodeURIComponent(species)).then(r => r.json()).then(data => {
        if (!data.success) return;
        breedData = data.data;
        breedData.species = species;
        if (!breedData.breeds.some(b => b.id === chosenBreed && b.unlocked)) chosenBreed = 'random';
        renderBreedPicker();
    }).catch(console.error);
}

// The breed chosen for a new pet
function selectedBreed() {
    return chosenBreed;
}

function renderBreedPicker() {
    const picker = document.getElementById('breedPicker');
    if (!picker || !breedData) return;
    picker.replaceChildren();
    const card = (id, art, name, badge, locked) => {
        const button = el('button', 'breed-card' + (id === chosenBreed ? ' chosen' : '') + (locked ? ' locked' : ''));
        button.type = 'button';
        button.setAttribute('aria-pressed', id === chosenBreed ? 'true' : 'false');
        const picture = el('span', 'breed-art');
        picture.innerHTML = art; // Drawn by GochiArt from fixed lists only
        button.append(picture, el('span', 'breed-name', name));
        if (badge) button.append(el('span', 'breed-badge', badge));
        button.addEventListener('click', () => {
            chosenBreed = id;
            renderBreedPicker();
        });
        return button;
    };
    picker.append(card('random', '<span class="breed-surprise">🎲</span>', 'Surprise me', '', false));
    breedData.tiers.forEach(tier => {
        const heading = el('div', 'breed-tier', tier.name + (tier.unlocked ? '' : ' 🔒'));
        picker.append(heading);
        breedData.breeds.filter(b => b.tier === tier.tier).forEach(b => {
            picker.append(card(b.id, GochiArt.draw({species: b.species, look: b.typical, size: 64, label: b.name}),
                b.name, '', !b.unlocked));
        });
    });
    renderBreedDetail();
    const preview = document.getElementById('createPreview');
    const chosen = breedData.breeds.find(b => b.id === chosenBreed);
    if (preview) {
        preview.innerHTML = chosen ? GochiArt.draw({species: chosen.species, look: chosen.typical, size: 140, state: 'happy', label: chosen.name})
            : '<span style="font-size: 100px">🐣</span>';
    }
}

// What the chosen breed is like, or what unlocks it
function renderBreedDetail() {
    const detail = document.getElementById('breedDetail');
    if (!detail || !breedData) return;
    detail.replaceChildren();
    const b = breedData.breeds.find(x => x.id === chosenBreed);
    if (!b) {
        detail.append(el('div', 'panel-text', 'A random breed from those you have unlocked. Every breed has its own looks, ' +
            'temperament and care needs.'));
        return;
    }
    detail.append(el('div', 'breed-title', b.name), el('div', 'panel-text', b.about),
        el('div', 'panel-text', 'Temperament: ' + b.temperament.toLowerCase() + '.'));
    if (b.lifespan !== 1) {
        detail.append(el('div', 'panel-text', 'Lifespan: about ' + Math.round(b.lifespan * 100) + '% of a typical ' +
            (b.species === 'dog' ? 'dog' : 'cat') + "'s."));
    }
    tendencyList(detail, b.tendency_details, 'No special care needs beyond the usual.');
    if (!b.unlocked) {
        const tier = breedData.tiers.find(t => t.tier === b.tier);
        const needs = tier.needs.map(a => (a.earned ? '✅ ' : '⬜ ') + a.icon + ' ' + a.name + ' (' + a.description.toLowerCase() + ')');
        detail.append(el('div', 'breed-locked', '🔒 Unlocks when your pets have earned: ' + needs.join('; ') + '.'));
    }
}

// Lists tendencies: what each does, and the real-world lesson
function tendencyList(parent, tendencies, none) {
    if (!tendencies || !tendencies.length) {
        parent.append(el('div', 'panel-text cloud-note', none));
        return;
    }
    const list = el('ul', 'tendency-list');
    tendencies.forEach(t => {
        const item = el('li');
        item.append(el('strong', '', t.name + (t.strength < 1 ? ' (milder: inherited from both parents)' : '') + ': '),
            t.effect + ' ', el('span', 'cloud-note', t.lesson));
        list.append(item);
    });
    parent.append(list);
}

// The About panel's breed section
function renderBreedAbout(info, name) {
    const box = document.getElementById('aboutBreed');
    if (!box || !info || !info.breed) return;
    box.replaceChildren();
    const b = info.breed;
    const lineage = info.lineage;
    const title = el('div', '');
    title.append(el('span', 'about-label', 'Breed: '), b.name + (b.mixed && lineage ? '' : ' · ' + b.temperament.toLowerCase()));
    box.append(title);
    if (!b.mixed || !lineage) box.append(el('div', 'cloud-note', b.about));
    if (info.hearing) {
        box.append(el('div', '', '🦻 ' + name + ' is ' + info.hearing + '. Train it with hand signals.'));
    }
    if (lineage && lineage.parents && lineage.parents.length) {
        const parents = lineage.parents.map(p => p.name + ' (' + p.breed_name +
            (p.parent_breeds && p.parent_breeds.length ? ': ' + p.parent_breeds.join(' × ') : '') + ')');
        const line = el('div', '');
        line.append(el('span', 'about-label', 'Parents: '), parents.join(' and '));
        box.append(line);
        if (b.mixed) {
            const breeds = [...new Set(lineage.parents.map(p => p.breed_name))];
            box.append(el('div', '', 'A mixed breed' + (breeds.length > 1 ? ': ' + breeds.join(' × ') : '') + '.'));
            const from = lineage.from || {};
            const byParent = {};
            Object.keys(from).sort().forEach(feature => (byParent[from[feature]] = byParent[from[feature]] || []).push(feature));
            Object.entries(byParent).forEach(([parent, features]) =>
                box.append(el('div', 'cloud-note', 'From ' + parent + ': ' + features.join(', ') + '.')));
            box.append(el('div', 'cloud-note', 'Mixed breeds inherit a breed\'s health tendency only if both parents ' +
                'have it, and then more mildly. Crossbreeds tend to have fewer inherited disorders, but they can still ' +
                'have any of their parents\' breeds\' problems, so checkups still matter.'));
        }
        (lineage.notes || []).forEach(n => box.append(el('div', 'breed-locked', n)));
    }
    const care = el('div', '');
    care.append(el('span', 'about-label', 'Breed care: '));
    box.append(care);
    tendencyList(box, info.tendencies, 'nothing beyond the usual.');
}

// A small drawing of a saved pet for the pet list
function savedPetArt(pet) {
    const holder = el('span', 'saved-pet-art');
    holder.innerHTML = GochiArt.draw({species: pet.species, look: pet.look, size: 44,
        state: pet.is_alive ? (pet.hibernating ? 'hibernating' : 'idle') : 'dead', label: pet.name});
    return holder;
}
