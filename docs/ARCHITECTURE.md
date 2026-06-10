# Gochi Architecture Documentation

## Overview

Gochi is built with a layered architecture that separates concerns and promotes maintainability, testability, and scalability.

## Architecture Layers

```
┌─────────────────────────────────────────────────┐
│          Presentation Layer (CLI/UI)            │
├─────────────────────────────────────────────────┤
│         Application Layer (cmd/gochi)           │
├─────────────────────────────────────────────────┤
│      Business Logic Layer (internal/*)          │
│  ┌──────────┬──────────┬──────────┬──────────┐ │
│  │   Core   │   AI     │  Social  │ Genetics │ │
│  ├──────────┼──────────┼──────────┼──────────┤ │
│  │ Biology  │   Data   │  Cloud   │ Security │ │
│  └──────────┴──────────┴──────────┴──────────┘ │
├─────────────────────────────────────────────────┤
│      Infrastructure Layer (pkg/*)               │
│  ┌──────────┬──────────┬──────────┬──────────┐ │
│  │  Types   │  Config  │  Logger  │ Security │ │
│  └──────────┴──────────┴──────────┴──────────┘ │
├─────────────────────────────────────────────────┤
│           Data Layer (SQLite/Files)             │
└─────────────────────────────────────────────────┘
```

## Core Components

### 1. Game Loop (internal/core/gameloop.go)

The game loop orchestrates all system updates and manages the game state.

**Key Responsibilities:**
- Pet lifecycle management
- Event system coordination
- Auto-save and backup scheduling
- Time management
- Statistics tracking

**Design Patterns:**
- **Game Loop Pattern**: Main update cycle running at configurable FPS
- **Observer Pattern**: Event system for decoupled communication
- **Singleton Pattern**: Central game state management

**Thread Safety:**
- Uses sync.RWMutex for concurrent access
- Safe for multiple goroutines reading/writing game state

### 2. Digital Pet (internal/core/pet.go)

Each digital pet is an autonomous entity with complex state.

**State Components:**
- Physical Stats: Health, hunger, happiness, energy, age
- Personality Matrix: Curiosity, sociability, playfulness, independence
- Memory Systems: Short-term and long-term interaction history
- Biological Systems: Metabolism, digestion, circadian rhythm

**Update Cycle:**
```
Update(deltaTime) → 
  UpdatePhysiological() → 
  UpdateBehavior() → 
  UpdateMemory() → 
  EmitStateChange()
```

### 3. Data Manager (internal/data/manager.go)

Centralized data persistence with caching and cloud sync.

**Architecture:**
```
DataManager
  ├── LocalStorage (File-based persistence)
  ├── PetCache (LRU cache with TTL)
  ├── BackupManager (Versioned backups)
  └── CloudSyncManager (Cloud synchronization)
```

**Features:**
- **Caching**: LRU cache with configurable size and TTL
- **Encryption**: Optional AES-256 encryption for sensitive data
- **Checksums**: Data integrity verification
- **Versioning**: Data format versioning for migration

**Backup Strategy:**
- Automatic periodic backups
- Configurable retention policy (max backups)
- Incremental backup support
- Backup verification with checksums

### 4. Authentication System (internal/cloud/auth_sqlite.go)

Production-ready authentication with SQLite backend.

**Security Features:**
- **Password Hashing**: PBKDF2 with 100,000 iterations
- **Salt Generation**: Cryptographically secure random salts
- **Rate Limiting**: 5 attempts per 15 minutes
- **Session Management**: Secure token generation with expiration
- **Audit Logging**: All authentication events logged

**Database Schema:**
```sql
users (
  id TEXT PRIMARY KEY,
  username TEXT UNIQUE,
  email TEXT UNIQUE,
  password_hash TEXT,
  password_salt TEXT,
  created_at INTEGER,
  last_login_at INTEGER
)

sessions (
  token TEXT PRIMARY KEY,
  user_id TEXT,
  created_at INTEGER,
  expires_at INTEGER,
  ip_address TEXT,
  device_id TEXT
)
```

### 5. Security Package (pkg/security/)

Comprehensive security utilities.

**Components:**

**Rate Limiter (ratelimit.go)**
- Token bucket algorithm
- Configurable limits and windows
- Automatic cleanup of stale entries
- Thread-safe concurrent access

**Input Validation (validation.go)**
- Username validation (3-20 chars, alphanumeric + _ -)
- Email validation (RFC 5322 simplified)
- Password strength checking (8+ chars, complexity requirements)
- Path traversal prevention
- Pet ID validation

**Audit Logging (audit.go)**
- Security event logging
- Structured log format
- Event types: login, registration, rate limiting, path traversal
- Contextual information (user, IP, timestamp)

### 6. Logger (pkg/logger/)

Structured logging with log/slog (Go 1.21+).

**Features:**
- Multiple output formats (text, JSON)
- Configurable log levels
- File output with multi-writer
- Source location tracking
- Graceful shutdown with log flushing

**Usage:**
```go
logger.Info("message", "key", value)
logger.Error("error occurred", "error", err, "context", ctx)
logger.Debug("debug info", "data", debugData)
```

### 7. Configuration (pkg/config/)

YAML-based configuration with environment variable overrides.

**Configuration Hierarchy:**
1. Default values
2. YAML configuration file
3. Environment variables (highest priority)

**Configuration Structure:**
```yaml
app:        # Application settings
log:        # Logging configuration
game:       # Game loop settings
auth:       # Authentication settings
cloud:      # Cloud sync settings
```

## Data Flow

### Pet Update Flow

```
User Interaction
      ↓
CommandProcessor
      ↓
GameLoop.ProcessInteraction()
      ↓
Pet.ProcessUserInteraction()
      ↓
  ┌───┴────┐
  │ Update │
  │  Pet   │
  └───┬────┘
      ↓
┌─────┴─────┐
│  Biology  │ → Metabolism, digestion, aging
│  Behavior │ → Personality, emotions, memory
│  Social   │ → Relationships, communication
└─────┬─────┘
      ↓
Event Emission
      ↓
Event Handlers
      ↓
  ┌───┴────┐
  │  Save  │ (if auto-save enabled)
  │  Pet   │
  └────────┘
```

### Authentication Flow

```
User Credentials
      ↓
AuthManager.Login()
      ↓
Rate Limiter Check
      ↓
Credential Validation
      ↓
Password Verification (PBKDF2)
      ↓
Session Creation
      ↓
Audit Logging
      ↓
Session Token (returned)
```

### Cloud Sync Flow

```
Local Pet Data
      ↓
DataManager.SavePet()
      ↓
LocalStorage.Save()
      ↓
CloudSyncManager.SyncAll()
      ↓
  ┌───┴────┐
  │Compare │
  │Versions│
  └───┬────┘
      ↓
Newer Data? → Upload/Download
      ↓
Cloud Storage Provider
      ↓
Sync Complete
```

## Concurrency Model

### Thread Safety Strategies

1. **Mutex-based Protection**
   - GameLoop: sync.RWMutex
   - DataManager: sync.RWMutex
   - AuthProvider: sync.RWMutex
   - RateLimiter: sync.RWMutex

2. **Immutable Data**
   - Configuration objects
   - Event payloads
   - Pet statistics snapshots

3. **Channel Communication**
   - Event system uses channels
   - Background task coordination
   - Shutdown signaling

### Goroutine Management

```
Main Goroutine
  ├── Game Loop Update (ticker-based)
  ├── Auto-save Worker (periodic)
  ├── Auto-backup Worker (periodic)
  ├── Signal Handler (shutdown)
  └── Event Processors (pool)
```

## Error Handling

### Error Handling Strategy

1. **Error Propagation**
   - Errors bubble up with context (fmt.Errorf with %w)
   - Structured logging at error origin
   - Caller decides recovery strategy

2. **Panic Recovery**
   - Event handlers have panic recovery
   - Game loop continues after recovered panics
   - Stack traces logged for debugging

3. **Graceful Degradation**
   - Cloud sync failures don't block local saves
   - Authentication failures have retry limits
   - File operations have fallback paths

### Error Logging Levels

- **Debug**: Detailed diagnostic information
- **Info**: Normal operational events
- **Warn**: Potential issues, degraded operation
- **Error**: Failures requiring attention

## Performance Optimizations

### 1. Caching Strategy

**LRU Cache with TTL:**
- Configurable size limit (MB)
- Time-to-live per entry
- Automatic eviction of expired entries
- Cache hit ratio tracking

### 2. Lazy Loading

- Pet data loaded on demand
- Relationship maps built incrementally
- Genetic data computed when needed

### 3. Batch Operations

- Batch pet updates in game loop
- Bulk database operations
- Aggregated event processing

### 4. Memory Management

- Object pooling for frequently allocated objects
- Careful goroutine lifecycle management
- Periodic cleanup of stale data

## Security Architecture

### Defense in Depth

```
Layer 1: Input Validation
  ↓
Layer 2: Rate Limiting
  ↓
Layer 3: Authentication & Authorization
  ↓
Layer 4: Data Encryption
  ↓
Layer 5: Audit Logging
```

### Security Controls

1. **Input Validation**
   - All user input validated before processing
   - Path traversal prevention
   - SQL injection prevention (parameterized queries)
   - XSS prevention (output encoding)

2. **Authentication**
   - Strong password requirements
   - Secure password hashing (PBKDF2)
   - Session token security
   - Rate limiting on auth attempts

3. **Data Protection**
   - Optional encryption at rest
   - Secure file permissions
   - Checksum verification
   - Backup integrity checks

4. **Audit Trail**
   - All security events logged
   - Structured log format
   - Timestamp and context included
   - Tamper-evident logging

## Testing Strategy

### Test Pyramid

```
       ┌─────────┐
       │   E2E   │     Integration Tests
       ├─────────┤
       │  Integ  │     Component Integration
      ┌┴─────────┴┐
     ┌┴───────────┴┐   Unit Tests
    ┌┴─────────────┴┐
    │   Unit Tests  │
    └───────────────┘
```

### Test Coverage Goals

- Unit Tests: 80%+ coverage
- Integration Tests: Key workflows
- Benchmark Tests: Performance regression
- Security Tests: Attack scenarios

### Test Organization

```
├── *_test.go          # Unit tests (alongside code)
├── test/
│   ├── integration/   # Integration tests
│   └── benchmark/     # Performance benchmarks
```

## Deployment Architecture

### Single Instance Deployment

```
┌─────────────────────────────────┐
│     Gochi Application           │
│  ┌──────────┐  ┌──────────┐    │
│  │  Game    │  │  Auth    │    │
│  │  Loop    │  │  System  │    │
│  └────┬─────┘  └────┬─────┘    │
│       │             │           │
│  ┌────┴──────┬──────┴─────┐    │
│  │  SQLite   │  File      │    │
│  │  Auth DB  │  Storage   │    │
│  └───────────┴────────────┘    │
└─────────────────────────────────┘
```

### Future: Distributed Architecture

```
┌──────────────┐     ┌──────────────┐
│  Client App  │────▶│  API Server  │
└──────────────┘     └──────┬───────┘
                            │
              ┌─────────────┼─────────────┐
              ↓             ↓             ↓
       ┌──────────┐  ┌──────────┐ ┌──────────┐
       │PostgreSQL│  │  Redis   │ │  S3/GCS  │
       │   Auth   │  │  Cache   │ │Cloud Save│
       └──────────┘  └──────────┘ └──────────┘
```

## Extension Points

### Adding New Pet Behaviors

1. Create behavior module in `internal/ai/behaviors/`
2. Implement `Behavior` interface
3. Register behavior in behavior registry
4. Add personality trait influences
5. Add tests

### Adding New Biological Systems

1. Create system in `internal/biology/`
2. Implement update cycle
3. Connect to existing systems (hormones, etc.)
4. Add UI indicators
5. Add tests

### Adding Cloud Providers

1. Implement `CloudProvider` interface
2. Add authentication mechanism
3. Implement retry logic
4. Add integration tests
5. Document provider-specific setup

## Best Practices

### Code Organization

- Keep packages focused and cohesive
- Minimize dependencies between packages
- Use interfaces for abstraction
- Document public APIs

### Error Handling

- Always check errors
- Add context to errors
- Log at appropriate levels
- Fail fast on initialization errors

### Testing

- Write tests before fixing bugs
- Test edge cases
- Use table-driven tests
- Mock external dependencies

### Security

- Validate all inputs
- Use parameterized queries
- Log security events
- Follow principle of least privilege

## Migration Guide

### Data Format Versioning

Current version: 1.0.0

**Version Check:**
```go
if petData.Version != CurrentDataVersion {
    return migrateData(petData)
}
```

**Migration Path:**
- 0.1.0 → 1.0.0: Add checksum field
- Future versions will include migration functions

## Monitoring and Observability

### Metrics to Track

- Pet update latency
- Save/load times
- Cache hit ratio
- Authentication success/failure rate
- Cloud sync latency
- Memory usage
- Goroutine count

### Logging Best Practices

- Use structured logging
- Include context (user ID, pet ID)
- Log at appropriate levels
- Don't log sensitive data (passwords, tokens)

### Health Checks

- Database connectivity
- File system access
- Cloud provider connectivity
- Memory usage within limits

## Future Enhancements

### Planned Features

1. **Multi-User Support**
   - Shared pet access
   - Permission system
   - Activity feed

2. **Real-time Sync**
   - WebSocket-based sync
   - Conflict resolution
   - Optimistic locking

3. **Advanced Analytics**
   - Pet health trends
   - Behavior patterns
   - Relationship graphs

4. **Plugin System**
   - Custom behaviors
   - Custom biological systems
   - Custom UI components
