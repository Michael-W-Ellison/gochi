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
│   ├── gochiserver/    # Web server and game (main application)
│   └── gochihub/       # The cloud hub on its own, without the game
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

# Build the game and the standalone hub (creates ./bin/gochi and ./bin/gochihub)
make build

# Run the application
make run
```

Then open http://localhost:8080 in a browser.

**Platforms**: the game runs on Windows, macOS and Linux. `make release` builds the game and the hub
for each (64-bit Intel/AMD and ARM) into `dist/`, one folder per platform with the `web/` and
`configs/` folders the program needs beside it. Phones and tablets (iOS and Android) play in their
browser, connected to a computer running the game on the same network; AR uses the phone's camera.

### Command-Line Options

Run the server directly to pass options, for example `./bin/gochi -autosave 30s` or
`go run ./cmd/gochiserver -data-dir ./my-saves`.

| Option | Default | Description |
|---|---|---|
| `-config` | `configs/config.yaml` | Settings file (see [Configuration](#configuration)); the default is used only if it exists |
| `-addr` | `127.0.0.1:8080` | Address and port the game listens on. The default answers only this computer; `:8080` lets paired phones and tablets on the network play too (see [Privacy](#privacy-and-parental-controls)) |
| `-data-dir` | `~/.gochi` | Directory for saved pets (`saves/`) and backups (`backups/`) |
| `-autosave` | `1m` | How often the active pet is saved automatically; `0` turns autosave off |
| `-away-pause` | `2m` | Pause time once the player has been away this long; `0` turns the away pause off |
| `-api-token` | (none) | Turn on the REST API at `/api/v1` for requests carrying this token (see [REST API](#rest-api)) |
| `-time-scale` | `game` | Game speed: `game` (one game day per real minute), `24x` (one game day per real hour), `4x` (one game day per six real hours) or `realtime` |
| `-hub` | off | Also run as a cloud hub (see [Cloud](#cloud-backup-sync-and-sharing)) |
| `-hub-join-code` | (none) | The code players need to register with the hub |
| `-hub-addr` | `:8443` | Address the hub listens on, always over HTTPS |
| `-tls-cert`, `-tls-key` | (none) | Serve the game (and the hub) over HTTPS with this certificate and private key (PEM files); set both |

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

Each save file also records the pet's **history** (the last 500 kinds of care it was given, how it
took them, and what happened to it on its own), its **achievements**, what it has **unlocked** (places
discovered, shop items, foods and toys in its pantry, tricks learned) and lifetime **statistics**,
along with the game speed and autosave interval it was played with.

**Achievements** are milestones a pet reaches: learning a first trick or every trick, finding a first
souvenir or a crystal, discovering every place, growing up, becoming a senior, living past a typical
lifespan, a best friend, a baby, 500 Friendship Points earned, and owning five shop items. They are
listed under **About** and announced under **Recently** when earned.

### Cloud: Backup, Sync and Sharing

A Gochi server can also run as a **cloud hub** that other Gochi servers connect to, so you can back up
your pets, play them on other devices, share achievements and meet at [AR playdates](#ar-playdates).
The hub is self-hosted: it is just another Gochi server, and nothing is sent anywhere else.

**What other players see**: players never choose a name. The hub gives each account a handle made of
a color, an animal and a number, such as "Blue Otter 42", and that is the only name other players
see. Your pets' names never leave your computer: other players see "Blue Otter 42's dog". Nothing a
player types is ever shown to another player; there is no chat or messaging of any kind. See
[Safety](#safety).

**Running a hub**: start a server with `-hub -hub-join-code SOMECODE` (or `hub.enabled` and
`hub.join_code` in the config file) and give the join code to the people who may register. The hub
listens on its own address, `-hub-addr` (`:8443` by default), separate from the game, and always uses
HTTPS (TLS 1.2 or later): with the certificate given by `-tls-cert` and `-tls-key` if there is one,
otherwise with its own certificate, made on first start and kept in `<data-dir>/hub/`. The hub prints
the certificate's fingerprint when it starts. Its data is kept in `<data-dir>/hub/`: accounts (with
account keys stored only as hashes), each player's encrypted pets, and the shared achievements. With
no join code, nobody new can register. The hub allows each player 120 requests a minute (requests
without an account, such as registering, 120 a minute from each address), 50 pets and 64 MB of saves
per account (4 MB per pet), and 10,000 accounts.

To run a hub as its own service, without the game, use `gochihub` (`make build` puts it in `bin/`):
`gochihub -join-code SOMECODE -dir /var/lib/gochi-hub -addr :8443`. It serves only `/hub/v1`, always
over HTTPS (with `-tls-cert` and `-tls-key`, or its own certificate), and keeps its data in `-dir`
(default `~/.gochi-hub`).

**Connecting**: press **☁️ Cloud**, enter the hub's address (`https://` is assumed; plain `http://` is
refused unless the hub is on this computer) and the join code, and press **Connect**. The hub gives
you a handle, and your pets are uploaded straight away. If the hub uses its own certificate rather
than one from a certificate authority, the game remembers the certificate when it connects and
refuses any other afterwards, so nobody can stand in for the hub later; the Cloud panel shows the
certificate's fingerprint, which a parent can compare with the one the hub printed. To connect
another device to the same account, press **Show account key** on the computer running the game (other
devices are never shown it) and enter that key on the other device instead of the join code. The
connection is kept in `<data-dir>/cloud.json`, readable only by you. Connections made by older
versions (with usernames, or over plain HTTP) are forgotten; connect again.

**Encryption**: your game encrypts each pet's save (AES-256-GCM, with a key made from your account
key) before uploading it, and decrypts it after downloading. The hub stores only a hash of your
account key, so its files alone, or a copy of them, can't be read. All the hub knows about a pet is a
random-looking key, its version and size, and when it was last played. Whoever runs the hub could
still change it to record account keys as they arrive, so use a hub run by someone you trust.

**Syncing** happens when the program starts (before your last pet is restored; if the hub can't be
reached within 15 seconds, the game starts with the saves on this device), every 5 minutes, when
you press **Sync now**, and when you exit. Everything but the cloud features works without a
network connection. For each pet:

- a pet that changed only on this device is uploaded, and one that changed only on the hub is
  downloaded (the previous local save is kept as a backup). Saving a pet that hasn't changed (for
  example while the game is paused) doesn't count as a change;
- a pet that changed on both is resolved in favour of the copy that was played most recently; the
  other stays as a backup on its side (the hub keeps one previous version of each pet);
- a pet new on the hub is downloaded into a new slot, and your active pet is reloaded if a newer copy
  was downloaded;
- deleting a pet removes it from the hub. Your other devices then report that it was deleted on
  another device and stop syncing it, but keep their copy until you delete it there too.

**Sharing achievements**: in the Cloud panel, turn on **Share my pets' achievements** to post your
pets' achievements to the hub as they are earned, for everyone on the hub to see. It is off until you
turn it on. The game sends only which achievement it was and your pet's species; the hub writes the
achievement's name and description from the game's own list, shown as, for example, "🐾 Blue Otter
42's cat: Good Pet". The panel shows recent achievements from everyone sharing on the hub.

### Privacy and Parental Controls

Press **🔒 Privacy**:

- **Learn when I usually play**: the game learns when you play to look ahead for you (see
  [Daily Goals and What's Coming Up](#daily-goals-and-whats-coming-up)). Turn it off to stop, and
  press **Erase what was learned** to forget it. It never leaves your computer.
- **Parent PIN**: a parent can set a PIN of 4 to 8 digits. Once it is set, the PIN is needed to
  connect to or disconnect from a cloud hub, see the account key, change what is shared, start or join
  a playdate, rename or delete a pet, unpair devices, and change these settings (the page asks for
  it). After 5 wrong PINs, no PIN is accepted for 5 minutes. The PIN is stored as a salted hash in
  `<data-dir>/parental.json`; it stops a child changing settings in the game, not someone who can edit
  the computer's files.
- **Play on a phone or tablet**: by default the game answers only the computer it runs on. To play on
  a phone or tablet on the same Wi-Fi (for example for AR), start the game with `-addr :8080`. Each
  device must then be paired: Privacy shows the address to open and a code (8 letters and digits,
  valid for 10 minutes); the device shows a pairing page until the code is entered. After 5 wrong
  codes, pairing stops for 5 minutes and the code is thrown away. Paired devices are remembered (as
  hashes in `<data-dir>/devices.json`); **Unpair all devices** forgets them.

Your saves, backups and settings in the data directory can be read only by your user account (on
macOS and Linux; older saves are made private as they are saved again).

### Safety

- **No personal information**: the game never asks for a name, email, age, location or anything else
  about the player. Hub accounts have a handle the hub chooses, and pet names (1 to 24 characters)
  stay on the player's computer. Pet IDs are random.
- **No way to pass messages**: there is no chat, and nothing a player types reaches another player.
  Everything another player sees comes from fixed lists (handles, species, how a pet looks,
  achievements) or is written by the game or the hub (what happened at a playdate). The hub refuses
  any request with fields it doesn't expect. The only things players share beyond those lists are a
  pet's position at a playdate (rounded to 5 cm) and how sociable it is, and when things happen.
- **Other web sites can't use the game**: the game answers only requests from its own page (JSON
  from the same origin, to `localhost` or an IP address), so a web page the player visits can't read
  or change anything, even by pointing its own name at the computer (DNS rebinding). Responses carry
  no CORS headers.
- **Other devices need pairing**: the game answers only this computer unless started on the network,
  and then only paired devices.
- **Parents can lock settings** with a PIN (see above).

Whoever runs a hub for children is responsible for meeting child-privacy laws such as COPPA; the game
collects no personal information and lets parents control what is shared.

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
Good weather for sleeping and a **Cozy Bed** make its sleep more restful. Pets have a daily rhythm:
they get sleepy at night (cats sleep from about 11 pm to 5 am, dogs from 10 pm to 6 am, by the
game's clock), tire faster in the dark, and fall asleep more easily then. A pet that stays awake too
long builds up a sleep debt, which makes it tire faster and learn more slowly until it has slept it
off. While asleep it goes through light, deep and REM sleep; deep sleep helps the body repair
itself.

Under its name, the page shows your pet's surroundings: the weather, temperature, season and time
of day. Each pet has its own weather and seasons, which move on with its own time (so they pause
when the game is paused). Hot or stormy weather and the seasons change how quickly your pet gets
hungry, thirsty and tired and can lift or lower its mood. They also change what it chooses to do on
its own: pets are livelier in spring and on fine days, and quieter in winter, in the heat, in rain
or snow and at night (cats perk up at dawn and dusk). Storms make pets, nervous ones most of all,
seek company, and cuddly pets look for warmth when it is cold. In poor weather, and in the dark, a
pet doesn't ask to go exploring and enjoys walks less. Exploring in bad weather is riskier the
worse the conditions are, and trips at night find fewer souvenirs. Each pet also has a favorite
season, from its personality (curious pets love spring, energetic ones summer, adaptable ones
fall, and cuddly, low-energy pets winter), shown under **About**: it is happier in that season and
a little less happy in the opposite one.

Below the bars, five smaller bars show needs that build up over time: **Exercise** (met by play,
hide and seek and exploring), **Company** and **Affection** (petting, grooming, treats and
training), **Learning** (training and play) and **Adventure** (exploring). A pet asks for help when
a need runs low, and needs left completely unmet make it unhappy. After each action the page shows what changed, for
example "Energy −10% · Happiness +20% · Stress −5%". A stat that is already full does not change.

### Daily Goals and What's Coming Up

Under the day's care checklist, the page shows **Today's goals**: three goals for the day, each
worth bonus Friendship Points when you complete it.

- one asks for the part of the daily routine you have given least over the past week (for
  example brushing, if you rarely brush your cat);
- one asks for an activity: playing, practicing tricks, exploring or, at the highest levels,
  teaching a new trick;
- one asks you to raise your pet's happiness to a level above where it is now.

Goals start at level 1 and adapt to how you are doing: completing two or three of a day's goals
makes the next day's goals one half-level harder (up to level 5), and trying but completing none
makes them half a level easier. Each goal earns 5 points per level (rounded), and the page announces
each goal as you complete it. Goals are kept with the pet's save, and a new set starts each
calendar day.

**Coming up** shows what your pet will ask for next if nothing changes, and when (for example "In
about 13 minutes: Mochi is thirsty!"). The game works this out by running a copy of your pet
forward in time, up to when you usually stop playing. To know that, the game learns when you play:
each hour of the week you use the page counts towards your usual times, recent weeks counting more
than older ones. After a few visits it looks ahead until the end of your usual playing time
(between 10 minutes and 2 hours), and otherwise 30 minutes, and the panel says when you usually
play (for example "weekday evenings"). This is kept in `<data-dir>/owner.json` and never leaves
your computer.

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

Under the organ bars, the page shows your pet's **age and life stage**, its **sleep** (awake,
drowsy, or the stage of sleep it is in, and any sleep debt) and its **hormones**, which respond to
how it lives:

- **Stress hormone** (cortisol) follows stress and spikes with frights, scolding, injuries and
  illness. High levels weaken the immune system, so a stressed pet falls ill more easily, and make
  sleep less restful. Affection brings it down.
- **Mood hormone** (serotonin) rises with exercise, company, daylight and a full stomach, and sets
  the mood your pet settles back to when nothing is happening.
- **Bonding hormone** (oxytocin) is released by petting, grooming and treats a pet enjoys, and
  calms it.
- **Growth hormone** is high in young pets and released in deep sleep. It speeds up healing and
  helps exercise build fitness.
- **Sleep hormone** (melatonin) rises in the dark.

**Growing up and growing old**: a pet is young for its first 60 game days (about an hour of play at
the default speed), an adult until 6,000 game days (about 100 hours) and then a senior. Young pets
are hungrier, learn tricks faster and gain fitness more easily; only adults can have babies. Seniors
slow down more and more as they near the end of their lives: their metabolism slows, they recover
less energy in sleep, lose fitness faster, fall ill more easily and learn more slowly. The page
announces when a pet grows up and when it becomes a senior, and the care guide has a page on caring
for an older cat or dog.

A typical pet lives about 9,000 game days, about 150 hours of play at the default speed. Cats live
a little longer than dogs; a pet's genes add or take up to 10%, and its care over its life (a running
record of its wellbeing) up to 15%. Hover over the age line to see your pet's care record. Near the
end the pet shows the warning sign **Very old and frail**, and it then dies of old age, unless gentle
mode is on (see below). A **Life-extension potion** from the shop (1,000 points; it unlocks once the
pet has earned 600 points over its life) adds about 1,500 game days (25 hours of play) each time;
press **Give life-extension potion** under **Body** to give one. A pet that is well cared for and
given a potion now and then can live on indefinitely.

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
- **What it has learned**: under **About**, "Lately becoming" says which way your recent care is
  moving its personality (for example "more playful"), and "Has learned you respond to" lists, for
  each of its moods, the behavior of its own that your attention has rewarded most (for example
  "when happy, playing"). When it misses you, a pet asks for the kind of attention it expects to
  enjoy most in its current mood, from how it has taken that care in that mood before.
- **On its own**: besides playing, seeking attention and sniffing around, a pet grooms itself when
  its coat needs it (cats far more than dogs, which keeps a cat cleaner and calms it) and an
  energetic pet that hasn't had enough exercise runs around, which builds a little fitness.
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

  Under **Spend time together** in the same panel, tick up to three of your other pets and choose
  **Play**, **Nap** or, if your pet is a dog, **Walk** (dogs only). Every pair in the group gets on
  according to their personalities, species and history: dogs enjoy a walk as a pack, close
  companions curl up for a nap together while strangers won't, cooperative (agreeable) pets get on
  better, and territorial cats get on worse the bigger the group.

  - **Signals**: the pets talk to each other the way real ones do: dogs play-bow, wag and lick, or
    sniff and circle warily; cats slow-blink, rub and touch noses, or watch and keep their
    distance. The result lists what each pair did.
  - **Pecking order**: each pet has a rank from its personality (outgoing, territorial, confident
    pets rank higher, and young ones lower). The result shows the order and who leads the pack or
    is top cat.
  - **Conflicts**: two pets that don't get along clash. If one clearly outranks the other, the
    lower one backs down after a growl or hiss and is a little stressed; if they are close in rank
    they scuffle, which stresses and frightens both and makes them rivals. The page explains how to
    handle a clash between cats, dogs, or a cat and a dog.
  - **Group happiness**: the result gives how happy the group is together: the pets' average
    happiness, lowered when they don't get along.

  Everyone in the group is saved with what happened: good company lifts their mood and calms them,
  a walk or play exercises them and a good nap rests them, and each pair's relationship records the
  time together. Your pet takes it as care, like a meeting. **About** shows, for each pet yours
  knows, what they last did together. A friendship or rivalry fades slowly once two pets have
  spent 100 game days (about 1 hour 40 minutes of play) apart; family bonds never fade below a
  certain level.

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
  about 20 game days (faster in summer and fall, when food is plentiful, and slower in winter), so
  going to the same place again and again finds less. Places differ in other ways too: the park and
  the city have the most animals and people around, so trips there do the most for your pet's need
  for company (and a sociable pet comes home happier, having made friends), while wilder places do
  more for its need for adventure. Every place carries some risk of parasites, lowest at the park
  and the farm and highest on the mountain; it builds up over visits (faster for a pet with a weak
  immune system) until the pet catches them and needs medical care. It needs energy, water and food to go,
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
Both pets must be the same species and adults (60 to 6,000 game days old), healthy (health at least 60%) and happy (happiness at
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
- **Health**: the life-extension potion (see Health and Learning Real Care).

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
already hibernating. Gentle mode pets also never die of old age: past their expected lifespan they
stay seniors.

### Deleting Pets

Press **Switch Pet**, then **Delete** next to the pet. Deleting asks twice: first to confirm, then
to type the pet's name. Deleting removes the pet's save file and all of its backups, so it cannot
be undone.

### Development

```bash
# Run tests (go test -short skips the twelve-hour play test)
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

# Build release folders for Windows, macOS and Linux in dist/
make release
```

**Adding a species**: everything that differs between kinds of pet is in one definition per
species: `internal/core/species_cat.go` and `species_dog.go`. A new kind of pet is a new file with a
`SpeciesDef` (its names, sprites, body, toilet habits, care lessons and guide, daily routine,
petting spots, supplies, reactions and social behavior) registered with `RegisterSpecies`. The game,
its shop and the page (which reads `/api/species`) pick it up; registration checks that the
definition is complete, and a test checks every registered species.

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

Each client (by IP address) may make 120 requests a minute (set with `server.api_rate_limit`;
0 turns the limit off). Requests that fail authentication count too, which limits guessing the
token. Responses carry `X-RateLimit-Limit` and `X-RateLimit-Remaining`; a client over the limit
gets status 429 with `X-RateLimit-Reset`, the time it may try again.

Care through the API follows the game's rules: a pet that turns something down answers with
status 409 and the reason. Pets can't be deleted through the API (403), because deleting asks for
the confirmations described above. The API works only on the player's own pets: it has no accounts,
social or messaging routes.

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
scene. A walk counts towards the pet's **Exercise** need: every 500 m counts as much as one play
session (at most two per walk) and uses a little energy. A sleeping, tired, hibernating or exploring
pet can't go for a walk.

To use AR on a phone or tablet while the game runs on a computer, start the game with `-addr :8080`
and pair the device (see [Privacy and Parental Controls](#privacy-and-parental-controls)).

### AR Playdates

Players connected to the same cloud hub (see [Cloud](#cloud-backup-sync-and-sharing)) can meet in
one place and see their pets together in AR, up to six at a time. Each player brings their active
pet from their own game.

1. One player opens **☁️ Cloud** and presses **🐾 Start a playdate**. AR opens, and the top of the
   screen shows the playdate's six-character code.
2. The others type the code under **AR playdate** and press **Join**.
3. Everyone agrees on a spot they can all see, such as the middle of a table, and taps it in their
   own AR view. That marks the playdate's center; each pet walks to its own place around it, and
   the other players' pets appear around it too, named by their owner's handle and species, such as
   "Blue Otter 42's dog".
4. Tap another player's pet to have yours play with it. How it goes is worked out like an
   introduction (both pets' sociability, their species and how earlier meetings went), and both
   pets remember it: your game records it for your pet, earning care points as usual, and the
   other player's game records it for theirs. The panel shows what has happened (written by the hub,
   such as "Blue Otter 42's dog and Teal Wren 17's cat played together happily!") and how your pet
   gets along with each of the others.

Closing AR leaves the playdate; a player whose game stops sending updates for 30 seconds is
removed, and a playdate ends when everyone has left. Switching to another pet also leaves it.
Each device places pets in its own view of the room, so pets appear around the center in the same
arrangement on every device but are not aligned to the same compass direction. The game tells the
hub only each pet's species, whether it is young, how sociable it is, why it can't play if it can't
(as a code, such as "asleep"), which of its species' pictures it shows, and its position relative to
the center (rounded to 5 cm); when two pets play, only how well it went, as numbers. The hub names
the pets and writes everything the players read, refuses anything else, and keeps playdates in
memory only.

| Route | Description |
|---|---|
| `GET /api/playdate` | The playdate you are at, as last updated: its code, the players and their pets, and what happened |
| `POST /api/playdate` | `{"action": "start"}`, `{"action": "join", "code": "..."}` or `{"action": "leave"}`; also takes `"look"`, which picture your pet shows (`idle`, `happy`, `sleeping` and so on) |
| `POST /api/playdate/update` | Send your pet's `{"position": {...}}` relative to the center (and `"look"`); returns the playdate. Send it about once a second |
| `POST /api/playdate/play` | `{"handle": "..."}`: your pet plays with that player's pet |

## Configuration

At startup the server reads `configs/config.yaml` from the project directory (the one holding
`web/`), or the file given with `-config`. The file holds the same settings as the command-line
options:

| Setting | Option | Default |
|---|---|---|
| `server.addr` | `-addr` | `"127.0.0.1:8080"` (this computer only) |
| `server.data_dir` | `-data-dir` | `""` (`~/.gochi`); a relative path is relative to the file |
| `server.api_token` | `-api-token` | `""` (REST API off) |
| `server.api_rate_limit` | | `120` REST API requests a minute per client (0 for no limit) |
| `hub.enabled` | `-hub` | `false` (also run as a cloud hub) |
| `hub.join_code` | `-hub-join-code` | `""` (registration closed) |
| `hub.addr` | `-hub-addr` | `":8443"` (always HTTPS) |
| `simulation.time_scale` | `-time-scale` | `"game"` |
| `simulation.autosave` | `-autosave` | `1m` |
| `simulation.away_pause` | `-away-pause` | `2m` |
| `server.tls_cert`, `server.tls_key` | `-tls-cert`, `-tls-key` | `""` (HTTP); relative paths are relative to the file |

A command-line option overrides the file, and a setting the file leaves out keeps its default.
The server stops with an error if the file has a setting it doesn't know or an invalid value, so
a misspelled setting is never silently ignored. The included file holds the defaults.

## Documentation

Detailed documentation can be found in the `docs/` directory:

- [Architecture Overview](docs/architecture.md)

## Development Status

**Current Version**: 0.1.0-alpha

This project is in early development.

## Performance

The simulation runs in the server, so the page only shows its state and stays responsive.

- **Memory**: the target is under 500 MB. A test plays twelve hours at the default speed (720 game
  days, with care every half minute): the server's heap stays under 2 MB, and a pet's save stops
  growing at under 1 MB, since a pet keeps its 1,000 strongest long-term memories and its last 500
  history entries. A server tick takes about 12 µs.
- **Battery**: the page updates every 2 seconds while charging, every 5 seconds on battery, and every
  15 seconds with animations off on low battery (20% or less), where the browser reports the battery
  (Chrome, Edge and other Chromium browsers). A hidden page doesn't update at all, and animations
  stop for players who ask their system for reduced motion.
- **Network**: scripts and styles are served with a version hash in their address and may be
  cached for a year, by the browser or by a proxy or CDN in front of the server; a new version gets
  a new address.
- **Platforms**: Windows, macOS and Linux (see [Installation](#installation)); iOS and Android
  through the browser.

## License

[License information to be added]

## Contributing

[Contribution guidelines to be added]

## Acknowledgments

Based on the Software Design Document for an Advanced Tamagotchi Digital Pet System.
