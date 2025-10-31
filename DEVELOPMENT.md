# diffwatch Development Plan

## Overview
A CLI tool that watches file changes in a directory and provides real-time diff visualization in the terminal.

## Architecture

```
┌─────────────────┐
│  File Watcher   │ (fsnotify)
└────────┬────────┘
         │ events
┌────────▼────────┐
│  Event Queue    │ (buffered channel)
└────────┬────────┘
         │
┌────────▼────────┐
│   Debouncer     │ (time-based batching)
└────────┬────────┘
         │
┌────────▼────────┐
│  State Manager  │ (file snapshots)
└────────┬────────┘
         │
┌────────▼────────┐
│  Diff Engine    │ (go-diff)
└────────┬────────┘
         │
┌────────▼────────┐
│   TUI Renderer  │ (bubbletea)
└─────────────────┘
```

## Tech Stack
- **Language**: Go 1.21+
- **File Watching**: fsnotify
- **TUI Framework**: bubbletea + lipgloss
- **Diff Algorithm**: sergi/go-diff or pmezard/go-difflib
- **CLI**: cobra (optional, start simple with flag package)
- **Syntax Highlighting**: alecthomas/chroma (future)

## Project Structure
```
diffwatch/
├── cmd/
│   └── diffwatch/
│       └── main.go           # Entry point
├── internal/
│   ├── watcher/
│   │   ├── watcher.go        # File system watching
│   │   └── debouncer.go      # Event debouncing
│   ├── state/
│   │   └── manager.go        # File state management
│   ├── diff/
│   │   └── engine.go         # Diff computation
│   └── ui/
│       ├── model.go          # Bubbletea model
│       ├── view.go           # Rendering
│       └── update.go         # Event handling
├── pkg/
│   └── config/
│       └── config.go         # Configuration structs
├── go.mod
├── go.sum
├── README.md
├── DEVELOPMENT.md
└── AGENTS.md
```

## Development Phases

### Phase 1: MVP ✅ Complete
- [x] Project setup and structure
- [x] Basic file watcher (single directory)
- [x] Simple debouncer (100ms window)
- [x] In-memory state management
- [x] Basic diff computation (unified format)
- [x] Simple TUI (file list + diff view)
- [x] CLI args: watch path

**Goal**: Watch a directory, show live diffs when files change

### Phase 2: Core Features (In Progress)
- [x] Recursive directory watching
- [ ] Ignore patterns (.gitignore style)
- [ ] Multiple diff formats (unified, split, inline)
- [ ] File filtering by extension
- [ ] Syntax highlighting
- [x] Better error handling (permission errors)

### Phase 3: Performance & Reliability
- [ ] Large file handling (streaming, chunked reads)
- [ ] Memory limits (LRU cache for snapshots)
- [ ] Backpressure handling
- [ ] Stress testing (1000+ files)
- [ ] Graceful degradation

### Phase 4: Advanced Features (In Progress)
- [ ] Configuration file support
- [ ] Export diffs (to file, clipboard)
- [ ] Search/filter in diffs
- [ ] History navigation
- [x] Binary file detection
- [ ] Symlink handling

### Phase 5: Polish
- [ ] Comprehensive tests
- [ ] Documentation
- [ ] Installation scripts
- [ ] CI/CD pipeline
- [ ] Release packaging

## Implementation Guidelines

### Error Handling
- Use explicit error returns
- Wrap errors with context: `fmt.Errorf("reading file %s: %w", path, err)`
- Log errors, don't crash - continue watching other files
- Graceful shutdown on critical errors

### Concurrency Patterns
- One goroutine for file watcher
- Worker pool for diff computation (runtime.NumCPU())
- Buffered channels (size: 100) for event queue
- Context for cancellation propagation

### Performance Considerations
- Use `sync.Pool` for diff buffers
- Minimize allocations in hot paths
- Profile with pprof before optimizing
- Benchmark critical paths

### Security
- Validate paths (no traversal outside watch dir)
- Respect file permissions
- Max file size limit (default: 10MB for diff)
- Resource limits (max watched files: 10,000)

## Testing Strategy
- Unit tests for each package
- Integration tests for full pipeline
- Table-driven tests for edge cases
- Benchmarks for diff engine
- Manual testing with real-world scenarios

## Next Steps
1. Initialize Go module
2. Create directory structure
3. Implement basic file watcher
4. Add simple TUI
5. Wire everything together
