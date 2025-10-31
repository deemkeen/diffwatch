# diffwatch

A real-time CLI tool for watching file changes and visualizing diffs in your terminal.

## Features

- Real-time file watching with platform-native notifications
- Live diff visualization with colored output
- Recursive subdirectory watching
- Debounced events to batch rapid changes
- Beautiful TUI built with Bubbletea
- Automatic permission error handling

## Installation

### From Source

```bash
go install github.com/hv11297/diffwatch/cmd/diffwatch@latest
```

### Build Locally

```bash
git clone https://github.com/hv11297/diffwatch.git
cd diffwatch
go build -o bin/diffwatch ./cmd/diffwatch
```

## Usage

Watch the current directory:
```bash
diffwatch
```

Watch a specific directory:
```bash
diffwatch -path /path/to/directory
```

Watch all subdirectories recursively:
```bash
diffwatch -path /path/to/directory -recursive
```

## Options

- `-path string` - Path to watch for changes (default: current directory)
- `-recursive` - Watch all subdirectories recursively (default: false)

## Controls

- `q` or `Ctrl+C` - Quit the application

## How It Works

```
File Change → fsnotify → Debouncer → State Manager → Diff Engine → TUI
```

1. **File Watcher** - Monitors files using platform-native APIs (inotify, FSEvents, etc.)
2. **Debouncer** - Batches rapid changes within a 100ms window
3. **State Manager** - Tracks file contents for comparison
4. **Diff Engine** - Computes unified diffs between versions
5. **TUI Renderer** - Displays colorized diffs in real-time

## License

MIT
