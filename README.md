# Gochi - Advanced Digital Pet System

An advanced Tamagotchi-style digital pet simulation system that implements complex biological processes, adaptive AI behaviors, and social interaction systems.

## Overview

Gochi is a sophisticated digital organism simulator featuring:

- **Realistic biological simulation** with interconnected physiological systems
- **Adaptive personality development** based on care history and interactions
- **Social networking** capabilities with other digital pets
- **Evolutionary mechanics** allowing trait inheritance and breeding
- **Machine learning-driven** behavioral adaptation
- **Cross-platform** support with cloud persistence

## Features

### Core Systems

- **Biological Simulation Engine**
  - Metabolism, digestion, and cellular aging
  - Cardiovascular, respiratory, digestive, and nervous systems
  - Hormonal system with stress hormones, growth factors, mood regulators
  - Circadian rhythm affecting all biological processes

- **Behavioral AI System**
  - Dynamic personality matrix with evolving traits
  - Reinforcement learning for user interaction adaptation
  - Multi-dimensional emotional state modeling
  - Long-term and short-term memory systems

- **Social Interaction System**
  - Pet-to-pet communication protocols
  - Relationship mapping (bonds, hierarchies, friendships)
  - Group dynamics and pack behavior

- **Genetic System**
  - Trait inheritance from parents
  - Genetic mutations and diversity
  - Breeding compatibility calculation

## Project Structure

```
gochi/
├── cmd/
│   └── gochiserver/    # Web server and game (main application)
├── internal/           # Private application code
│   ├── core/           # Core pet logic
│   ├── biology/        # Biological simulation systems
│   ├── ai/             # AI, memory and behavior systems
│   ├── social/         # Social interaction systems
│   ├── genetics/       # Evolutionary and genetics systems
│   ├── data/           # Data persistence and cloud sync
│   ├── simulation/     # Time management and needs systems
│   ├── interaction/    # User interaction processing
│   ├── environment/    # Environmental simulation
│   └── ar/             # Augmented reality support
├── pkg/
│   └── types/          # Shared type definitions
├── api/                # REST API package (served at /api/v1 with -api-token)
├── configs/            # Configuration files
├── web/                # Web UI assets
├── test/               # Integration tests
└── docs/               # Documentation
```

## Getting Started

### Prerequisites

- Go 1.22 or higher
- Make (optional, for using Makefile commands)

### Installation

```bash
# Clone the repository
git clone https://github.com/Michael-W-Ellison/gochi.git
cd gochi

# Install dependencies
make install

# Build the application (creates ./bin/gochi)
make build

# Run the application
make run
```

Then open http://localhost:8080 in a browser.

### Command-Line Options

Run the server directly to pass options, for example `./bin/gochi -autosave 30s` or
`go run ./cmd/gochiserver -data-dir ./my-saves`.

| Option | Default | Description |
|---|---|---|
| `-config` | `configs/config.yaml` | Settings file (see [Configuration](#configuration)); the default is used only if it exists |
| `-addr` | `:8080` | Address and port the web server listens on |
| `-data-dir` | `~/.gochi` | Directory for saved pets (`saves/`) and backups (`backups/`) |
| `-autosave` | `1m` | How often the active pet is saved automatically; `0` turns autosave off |
| `-away-pause` | `2m` | Pause time once the player has been away this long; `0` turns the away pause off |
| `-api-token` | (none) | Turn on the REST API at `/api/v1` for requests carrying this token (see [REST API](#rest-api)) |
| `-time-scale` | `game` | Game speed: `game` (one game day per real minute), `24x` (one game day per real hour), `4x` (one game day per six real hours) or `realtime` |

Durations use Go syntax, such as `30s`, `5m` or `1h`.

## Playing

### Saving and Loading Pets

Each pet has its own save file in `<data-dir>/saves/`. The game saves the active pet:

- automatically, every `-autosave` interval
- when you press **Save**
- when you switch to another pet or create a new one
- when you exit the program with **Ctrl+C**

The previous five saves of each pet are kept in `<data-dir>/backups/`. When the program starts, it
restores the most recently saved pet. If a save file is damaged, the newest working backup is used
instead.

To play with a different pet, press **Switch Pet**. The pet list shows every saved pet; press **Load**
to switch to one, or create a new pet below the list. Your current pet is saved first, so no
progress is lost.

### Pausing Time

Pets need regular care, so time only passes while you are playing:

- **Away pause**: if you have not used the page for `-away-pause` (2 minutes by default), time
  stops automatically and the page shows "Time paused while you were away." Time resumes when you
  come back and move the mouse, press a key or touch the screen. Leaving the browser tab open does
  not count as playing, so a pet is safe while you are at school or on a family trip.
- **Pause button**: press **Pause** before you leave to stop time until you press **Resume**.
- **Program closed**: no time passes while the program is not running. A restored pet continues
  exactly where it was saved.

Only the active pet's time passes; pets that are not loaded wait in their save files.

### Caring for Your Pet

When you create a pet, choose a **cat** or a **dog**. The game teaches the real care each one needs.

Use **Feed**, **Drink**, **Pet**, **Play**, **Groom** and **Heal** to look after your pet, **🧹 Scoop
litter** (cats) or **🧹 Clean up** (dogs) to keep its living area clean, and **🦮 Walk** to take a dog
outside. The bars show its health, energy, happiness, hunger and water.

Pets don't accept every kind of care at every moment: a full pet isn't hungry, a clean pet doesn't
need grooming, a very tired pet won't play, and a stressed or tired pet won't train. When your pet
turns something down nothing happens and no points are earned. Hover over a care button to see
what it does.

When your pet gets very tired it falls asleep. A sleeping pet only accepts food, water and medical
care; it sleeps until it is rested, recovering energy and losing tiredness, and wakes on its own.
Good weather for sleeping and a **Cozy Bed** make its sleep more restful.

Under its name, the page shows your pet's surroundings: the weather, temperature, season and time
of day. Each pet has its own weather and seasons, which move on with its own time (so they pause
when the game is paused). Hot or stormy weather and the seasons change how quickly your pet gets
hungry, thirsty and tired and can lift or lower its mood. Exploring in storms or extreme
temperatures is riskier, and trips at night find fewer souvenirs. Each pet also has a favorite
season, from its personality (curious pets love spring, energetic ones summer, adaptable ones
fall, and cuddly, low-energy pets winter), shown under **About**: it is happier in that season and
a little less happy in the opposite one.

Below the bars, five smaller bars show needs that build up over time: **Exercise** (met by play,
hide and seek and exploring), **Company** and **Affection** (petting, grooming, treats and
training), **Learning** (training and play) and **Adventure** (exploring). A pet asks for help when
a need runs low, and needs left completely unmet make it unhappy. After each action the page shows what changed, for
example "Energy −10% · Happiness +20% · Stress −5%". A stat that is already full does not change.

### Health and Learning Real Care

Your pet has a body that works like a real animal's. Under **Body**, four bars show its organ
systems:

- **Digestion**: food passes through the stomach and becomes waste. Eating too much at once, for
  example many treats in a row, makes the pet throw up and upsets its stomach; an upset gut takes in
  less of each meal.
- **Heart**: kept fit by play, exploring and walks, and fades without exercise. An unfit pet tires
  quickly.
- **Lungs**: harmed by foul air from a dirty living area, and recover in fresh air. Weak lungs also
  make a pet tire quickly.
- **Nerves**: worn down by long-term stress, and restored by calm. A worn nervous system makes the
  pet more easily stressed.

An organ system that gets too weak costs the pet health.

**The living area** gets dirty as the pet relieves itself. A cat uses its litter box several times
a day, so scoop it regularly; if the box gets too dirty, the cat goes elsewhere and gets stressed. A
dog goes outside on walks, so its home stays clean as long as it is walked often enough; a dog left
too long has an accident in the house.

**Illness**: a dirty living area makes the air foul and can make the pet ill: a cat can get a
urinary tract infection, a dog intestinal parasites, and damaged lungs can lead to a respiratory
infection. An illness gets worse until you give medical care (**Heal**), costing the pet health. Left
untreated it can kill the pet, or send it into hibernation in gentle mode. Treatment cures it, but it
can come back if the living area stays dirty.

The game teaches care in four ways:

- **Lessons**: after each action, a 💡 line explains why it matters for a real cat or dog, for
  example why a clean litter box prevents urinary problems or why you pick up after a dog.
- **Warning signs**: under **Body**, the pet shows signs before problems get serious (a smelly
  litter box, sneezing, tiring quickly, waiting by the door, straining in the litter box) with what
  each means and what to do. Good news, such as a recovery, is shown in green.
- **Care guide**: press **📖 Care guide** for short lessons on feeding, water, the litter box or
  bathroom breaks, exercise, grooming, vet care, breathing and stress. Topics related to the pet's
  current warning signs are marked ⚠️, and **Learn more** on a warning sign opens its topic.
- **Today's care**: a checklist of a real owner's daily routine, ticked off as you give care. For
  a cat: two meals, fresh water, scooping the litter box, play and brushing; for a dog: two meals,
  fresh water, three walks and play. It follows the real calendar day, not game time, so it shows
  the routine a real pet needs whatever the game speed.

The lessons and guide give general care advice. For a real pet, follow your veterinarian's advice.

**Recently**, under the checklist, lists what happened to your pet on its own while time passed:
when it got hungry or thirsty, showed a new warning sign, fell ill or recovered, came home from a
trip, got rusty at a trick, went into hibernation or died. It keeps the last few of these, so you
can see what happened while you weren't watching. Care you give is reported when you give it and
is not listed.

### Getting to Know Your Pet

- **What it is doing**: the line under the picture says what your pet is doing, such as
  "Biscuit is playing by itself", "Biscuit wants your attention", "Biscuit is out exploring"
  (only during a trip) or "Biscuit is hiding somewhere" (during hide and seek). What a pet
  chooses to do depends on its personality, on what it has learned it enjoys, and on how you have
  responded when it did the same thing before.
- **Learning from you**: when you respond to what your pet is doing, it learns from it. Petting a
  pet that wants your attention teaches it that seeking attention is rewarded, so it does it more;
  scolding a pet while it plays teaches it to play less. What it learns depends on its mood at the
  time, and it builds up gradually.
- **Requests**: your pet asks for what it needs in a speech bubble (a drink, food, grooming,
  attention or play), the matching button is highlighted, and the browser tab title shows 💬.
  A pet left alone for a while asks for the kind of attention it likes best.
- **About**: shows your pet's personality, how much it enjoys attention, what it likes, puts up
  with and dislikes, how its personality has changed since you adopted it, the treasures it
  has found, and its favorite season.
- **Attention**: by nature, some pets love attention, some tolerate it in moderation and some
  prefer their own space. Affectionate, outgoing pets love it; independent and nervous ones would
  rather be left alone; dogs are on the whole keener than cats. A pet that is lonely welcomes
  attention more. Too much petting or grooming in a short time overstimulates any pet. Attention a
  pet doesn't want (petting, grooming, play, training or walks it is reluctant about) stresses it
  instead of pleasing it, and the page explains the signs that a cat or dog has had enough.
- **Likes and dislikes**: a pet learns them from how it takes each kind of care, which depends on
  its nature and when the care is given, not just on the care itself. A pet that loves attention
  learns to like being petted; one that prefers its own space learns to dislike it, however often
  it is petted; one that tolerates it learns to put up with it. Care a pet has had five or more
  times but neither likes nor dislikes is listed under "Puts up with". Care repeated in quick
  succession counts less towards what it learns.
- **A personality that grows**: a pet's personality changes slowly with how it is treated.
  Enjoyed play makes it more playful, enjoyed petting more affectionate, good trips more curious,
  and consistent care calmer and more attached to you. Scolding, injuries, illness and accidents
  make it more anxious, and a pet left lonely or neglected becomes more independent, more anxious
  and less attached. Each experience changes it only a little, so changes show after many
  experiences (for example, about 30 enjoyed play sessions to become noticeably more playful).
  The same care repeated in quick succession counts less each time towards personality and
  learning (the second in a row counts half, the third a third, and so on, recovering over a few
  minutes), so 40 quick clicks change a pet about as much as 4 or 5 visits spread out. Repeated
  care still feeds, waters and cheers up your pet as usual; after a few repeats the page says that
  more of it right away won't change how your pet grows.
  Personality changes are not passed on to babies, which inherit their parents' genes.
- **Reactions**: each pet reacts in its own way. A hungry pet gobbles up its food while a full one
  barely eats; an affectionate pet loves being petted while an independent one wants space; a pet
  that has learned to love something enjoys it even more. How much your pet enjoys an action
  changes how much good it does (from 0.6 to 1.4 times the usual effect).

### Ways to Care

Several kinds of care come in variants, each with its own effect and lesson:

- **Feed ▾**: choose a food (or a treat) from the pantry.
- **Play ▾**: play together, or with a toy. Each toy suits a species and a kind of play (chasing,
  hunting, tugging, chewing, puzzles), giving more exercise, more mental stimulation or more
  calm.
- **Pet ▾**: choose where to pet. Most cats like their cheeks, chin and head and dislike belly rubs;
  most dogs like their chest and a scratch behind the ears, many dislike having their paws handled,
  and a pat on top of the head can feel threatening. Personality matters: affectionate, calm pets
  accept sensitive spots, and nervous ones less so. A cuddly, calm dog may love a belly rub.
- **🎓 Train**: choose a training method: clicker and reward (fastest, needs a clicker and treats),
  treat rewards (needs treats), praise (works best for pets that love attention) or scolding (slow,
  and makes the pet more anxious).
- **🏠 Enrich**: change something at home: for a cat a cardboard box, a window perch, food to
  hunt for, a scratching post or a toy swap; for a dog a treat-sniffing game, new things to sniff, a
  quiet den, a frozen food toy or a toy swap. Each keeps the pet's mind busy in its own way, and
  hiding places calm nervous pets. The same change done again within a few days feels less new.
- **🤝 Meet**: introduce your pet to another of your saved pets. How it goes depends on both
  personalities (sociable, easy-going pets get on; territorial, nervous ones clash), their
  species (dogs usually get on, cats are territorial, and a cat and a dog need the most care) and
  how earlier meetings went. Good meetings build a friendship and bad ones a rivalry, recorded for
  both pets and shown under **About**. The page explains how to introduce cats, dogs, or a cat and
  a dog.

### Activities

- **🎓 Train**: teach tricks (Sit, Shake a paw, Spin, Roll over, Play dead). Each session
  improves the trick; harder tricks take longer, and clever pets learn faster. Once a trick is
  learned, press **Do it!** to have your pet perform it. A tired pet can't train. Tricks need
  practice to stay sharp: a trick that hasn't been trained or performed for 120 game days (two
  hours of play at the default speed) starts to fade. A learned trick that fades becomes
  **rusty**; your pet can't perform it until you **Practice** it, which takes a session or two,
  because it never forgets more than a quarter of a trick it has learned. Progress on a trick that
  was never learned fades away completely.
- **🙈 Hide & Seek**: your pet hides in one of four places and you get two guesses. Finding it
  counts as play and earns a 5-point game bonus (at most once every 3 minutes). Press **Stop
  playing** (or close the panel) to end the game early; an unfinished game earns nothing.
- **🧭 Explore**: send your pet out for about 2 minutes. Once it has discovered more than the
  park, pick where it goes: the park, then the beach (discovered after 2 trips), the farm (4),
  the forest (6), the city (9) and the mountain (12). Wilder places hold more to find and are more
  dangerous. What your pet finds is taken from the place (a souvenir takes more than looking
  around and finding nothing), shown as a bar of what is left to find; a place grows back over
  about 20 game days, so going to the same place again and again finds less. It needs energy, water and food to go,
  and can't be cared for while it is away, though you can still clean its living area. Pausing
  time pauses the trip too. Trips don't always go well:
  - a little over half the time your pet brings home a souvenir for its treasure collection (curious
    pets find souvenirs, including rare crystals, more often);
  - about a third of the time it finds nothing of value, though the trip is still fun;
  - about one trip in nine it gets hurt: it loses health, comes home frightened and becomes warier
    of exploring. An injured pet asks to be healed and can't explore again until you press
    **Heal**. Adaptable pets get hurt less often. A badly hurt pet can die, or hibernates if gentle mode is on.

  Press **Call home early** to bring your pet back before the trip ends; it comes home with
  nothing from the trip. A pet that enjoys exploring sniffs around the house on its own; it only
  goes on a trip when you press **Explore**.

### Breeding

Press **💞 Breed** to have a baby with one of your other saved pets. The panel lists each saved
pet with how genetically compatible it is with your pet and, if they can't breed right now, why.
Both pets must be the same species and at least 7 game days old, healthy (health at least 60%) and happy (happiness at
least 50%), and at least 30% compatible; neither can be hibernating, exploring or have passed
away. Breeding can still fail by chance; less compatible pairs fail more often. The season matters
too: babies are likelier in spring and less likely in fall and winter, and the panel says so. Two
pets can have at most 5 babies together.

The baby is saved as a new pet (find it under **Switch Pet**); your current pet stays active. Its
personality is inherited from both parents through their genes, with an occasional mutation, and
the **About** panel shows its generation. After having a baby, each parent must wait 3 game days
before breeding again.

### Friendship Points and the Shop

Caring for a pet earns it Friendship Points: 5 points for each kind of care (including training
and exploring), at most once every 3 minutes per kind (more clicks still help your pet but earn nothing extra). Points belong to the
pet that earned them. Press **Shop** to spend them on items for that pet:

| Item | Price | Effect |
|---|---|---|
| Sturdy Food Bowl | 100 | Hunger builds 20% slower |
| Automatic Feeder | 250 | Hunger builds 40% slower |
| Big Water Bowl | 100 | Thirst builds 20% slower |
| Water Fountain | 250 | Thirst builds 40% slower |
| Squeaky Toy | 80 | Playing is 25% more fun |
| Puzzle Ball | 200 | Playing is 50% more fun |
| Cozy Bed | 150 | Tiredness builds 25% slower |
| Soft Brush | 80 | Grooming is 50% more effective |
| Training Clicker | 40 | Needed for clicker training, which is 40% faster |
| Balloon, Potted Plant, Night Lamp, Painting, Rainbow Mural | 30–120 | Decorations |

Items work automatically once bought; if a pet owns two items of the same kind, the better one
applies. Owned items are shown under the pet's name, with an **Active effects** list of the items
currently working. When a toy or brush improves an action, the page says so after that action.
Toys make play more fun without making it more tiring.

**Supplies** are sold in the same shop and are used up: food, treats and toys that wear out. They
are kept in the pet's pantry, and the shop only shows those for the pet's species.

- **Food**: each species' dry food is always available and free, so a pet never goes hungry for
  lack of points. Wet food adds water, plain cooked fish or chicken is a tasty topper (not a
  complete meal), and premium food is easy to digest.
- **Foods to avoid**, sold so you can learn what not to feed: cow's milk (most adult cats and many
  dogs are lactose intolerant), table scraps (fatty and sometimes toxic), and chocolate for dogs.
  The page warns you before feeding them. Milk and table scraps upset the stomach; chocolate
  poisons a dog, which then needs medical care (**Heal**) before it gets worse.
- **Treats**, used for treat and clicker training.
- **Toys**: for cats a feather wand, a catnip mouse (about two cats in three react to catnip,
  depending on their genes) and a laser pointer; for dogs a tennis ball, a rope toy and a chew toy;
  and a puzzle feeder for either. Most toys wear out after a number of play sessions.

Supplies come in rarities. Common ones are available from the start; uncommon, rare, epic and
legendary ones unlock once the pet has earned 50, 150, 300 and 600 Friendship Points over its
life. Two supplies are on sale (25% off) each day.

### Hibernation (Gentle Mode)

By default a pet that is neglected until its health or water runs out, or its food and energy
both do, dies. The pet screen then says how it died, and you can switch to or create another pet.

Players who prefer a gentler game can turn on **Gentle mode**, either when creating a pet or later
with the checkbox on the pet screen. With gentle mode on, a pet whose health or water falls to a
critical level (10%), or whose food and energy both do, goes into hibernation instead of dying.
While it hibernates its stats, age and mood stay exactly as they are and it does not respond to
care. Press **Revive** to wake it: its health, water, food and energy are restored to at least
half, and you can look after it again. Turning gentle mode off does not wake a pet that is
already hibernating.

### Deleting Pets

Press **Switch Pet**, then **Delete** next to the pet. Deleting asks twice: first to confirm, then
to type the pet's name. Deleting removes the pet's save file and all of its backups, so it cannot
be undone.

### Development

```bash
# Run tests
make test

# Run benchmarks (pet creation, updates, saving and loading, server ticks)
go test -run '^$' -bench . ./internal/core ./cmd/gochiserver

# Run tests with coverage
make test-coverage

# Format code
make fmt

# Run linter
make lint

# Run in development mode with live reload
make dev
```

## REST API

Other programs can use the REST API from the `api` package when the server is started with
`-api-token`, for example `./bin/gochi -api-token my-secret`. Requests to `/api/v1/...` must send
the header `Authorization: Bearer my-secret`; `/health` and `/ready` need no token. Without
`-api-token` these routes are off.

| Route | Description |
|---|---|
| `GET /api/v1/pets` | All saved pets |
| `POST /api/v1/pets` | Create a pet (`{"name": "...", "species": "cat", "randomize_traits": true}`; species is `cat` (default) or `dog`); it is saved without becoming the active pet |
| `GET /api/v1/pets/{id}`, `GET /api/v1/pets/{id}/status` | One pet |
| `PUT /api/v1/pets/{id}` | Rename a pet (`{"name": "..."}`) |
| `POST /api/v1/pets/{id}/interactions` | Care for a pet (`{"type": "feed", "intensity": 1}`; types: feed, drink, pet, play, train, groom, medical, reward, discipline, clean, walk) |
| `POST /api/v1/pets/{id}/interactions/feed`, `.../play` | Shortcuts for feeding and playing |
| `GET /api/v1/social/friends?pet_id=...` | A pet's friends |

Each client (by IP address) may make 120 requests a minute (set with `server.api_rate_limit`;
0 turns the limit off). Requests that fail authentication count too, which limits guessing the
token. Responses carry `X-RateLimit-Limit` and `X-RateLimit-Remaining`; a client over the limit
gets status 429 with `X-RateLimit-Reset`, the time it may try again.

Care through the API follows the game's rules: a pet that turns something down answers with
status 409 and the reason. Pets can't be deleted through the API (403), because deleting asks for
the confirmations described above. Social features that need other players (nearby pets, friend
requests) and cloud sync answer with an error, because the server has no cloud service.

## Augmented Reality

Press **📷 See in your room (AR)** to see your pet through the camera. On a phone or tablet whose
browser supports WebXR augmented reality (for example Chrome on Android), point the camera at the
floor or a table until a surface is found, then tap to place your pet. On other devices the pet
appears in a drawn room instead. In both:

- tap the pet to pet it, or use the buttons to feed it or play with it; this is the same care as
  the buttons on the main screen, with the same rules, reactions and Friendship Points;
- tap elsewhere to send your pet there, or 👋 to call it over;
- the top of the screen shows what your pet is doing, and its animation follows it;
- 🏃 tracks a walk with your pet (only with WebXR, which can follow the device's position); when
  you stop, the distance counts towards its **Exercise** need;
- ✕ closes AR.

The page (`web/static/js/ar.js`, connected through `web/static/js/ar-link.js`) uses these
endpoints, and other AR clients can use them too:

| Route | Description |
|---|---|
| `GET /api/ar` | The scene: session state, surfaces, the placed pet (with its current behavior) and walk tracking |
| `POST /api/ar/session` | `{"action": "start"}`, `"pause"`, `"resume"` or `"stop"` |
| `POST /api/ar/plane` | Report a surface (`{"plane": {"id": "floor", "type": "horizontal", "center": {...}, "extent": {...}}}`) or `{"remove": "floor"}` |
| `POST /api/ar/camera` | Report the camera's `{"transform": {...}}` and an optional `{"light": {...}}` estimate |
| `POST /api/ar/place` | Place the active pet at `{"position": {...}}`, or in the middle of the largest floor or table |
| `POST /api/ar/gesture` | A gesture (`{"type": "tap", "position": {...}}`). On the pet, a tap pets it, a double tap plays, a long press feeds it and a swipe throws a ball |
| `POST /api/ar/exercise` | `{"action": "start"}`, `{"action": "position", "position": {...}}` while walking, then `{"action": "stop"}` |

Gestures follow the same rules as the care buttons. Stopping the session removes the pet from the
scene. A walk counts towards the pet's **Exercise**
need: every 500 m counts as much as one play session (at most two per walk) and uses a little
energy. A sleeping, tired, hibernating or exploring pet can't go for a walk.

## Configuration

At startup the server reads `configs/config.yaml` from the project directory (the one holding
`web/`), or the file given with `-config`. The file holds the same settings as the command-line
options:

| Setting | Option | Default |
|---|---|---|
| `server.addr` | `-addr` | `":8080"` |
| `server.data_dir` | `-data-dir` | `""` (`~/.gochi`); a relative path is relative to the file |
| `server.api_token` | `-api-token` | `""` (REST API off) |
| `server.api_rate_limit` | | `120` REST API requests a minute per client (0 for no limit) |
| `simulation.time_scale` | `-time-scale` | `"game"` |
| `simulation.autosave` | `-autosave` | `1m` |
| `simulation.away_pause` | `-away-pause` | `2m` |

A command-line option overrides the file, and a setting the file leaves out keeps its default.
The server stops with an error if the file has a setting it doesn't know or an invalid value, so
a misspelled setting is never silently ignored. The included file holds the defaults.

## Documentation

Detailed documentation can be found in the `docs/` directory:

- [Architecture Overview](docs/architecture.md)

## Development Status

**Current Version**: 0.1.0-alpha

This project is in early development.

## Performance Targets

- **Frame Rate**: 60 FPS
- **Memory Usage**: < 500 MB RAM
- **Battery Optimization**: Adaptive processing based on battery level
- **Cross-Platform**: iOS, Android, Windows, macOS

## License

[License information to be added]

## Contributing

[Contribution guidelines to be added]

## Acknowledgments

Based on the Software Design Document for an Advanced Tamagotchi Digital Pet System.
