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
├── api/                # REST API package (not yet used by the server)
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
| `-addr` | `:8080` | Address and port the web server listens on |
| `-data-dir` | `~/.gochi` | Directory for saved pets (`saves/`) and backups (`backups/`) |
| `-autosave` | `1m` | How often the active pet is saved automatically; `0` turns autosave off |
| `-away-pause` | `2m` | Pause time once the player has been away this long; `0` turns the away pause off |

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

Use **Feed**, **Drink**, **Pet**, **Play** and **Groom** to look after your pet. The bars show its
health, energy, happiness, hunger and water.

### Friendship Points and the Shop

Caring for a pet earns it Friendship Points: 5 points for each kind of care, at most once every
3 minutes per kind (more clicks still help your pet but earn nothing extra). Points belong to the
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
| Balloon, Potted Plant, Night Lamp, Painting, Rainbow Mural | 30–120 | Decorations |

Items work automatically once bought; if a pet owns two items of the same kind, the better one
applies. Owned items are shown under the pet's name.

### Hibernation

If a pet's health or water falls to a critical level, or its food and energy both do, the pet goes
into hibernation instead of dying. While it hibernates its stats, age and mood stay exactly as they
are and it does not respond to care. Press **Revive** to wake it: its health, water, food and
energy are restored to at least half, and you can look after it again.

### Deleting Pets

Press **Switch Pet**, then **Delete** next to the pet. Deleting asks twice: first to confirm, then
to type the pet's name. Deleting removes the pet's save file and all of its backups, so it cannot
be undone.

### Development

```bash
# Run tests
make test

# Run tests with coverage
make test-coverage

# Format code
make fmt

# Run linter
make lint

# Run in development mode with live reload
make dev
```

## Configuration

The server is configured with the command-line options above. `configs/config.yaml` describes
planned settings (simulation parameters, performance targets, biological parameters, AI settings
and social features) but is not read by the server yet.

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
