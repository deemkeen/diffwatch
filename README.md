# diffwatch

A real-time CLI tool for watching file changes and visualizing diffs in your terminal.

## Features

- 🔍 **Real-time file watching** - Uses platform-native file system notifications
- 📊 **Live diff visualization** - See changes as they happen with colored diff output
- ⚡ **Debounced events** - Intelligently batches rapid file changes
- 🎨 **Beautiful TUI** - Built with Bubbletea for a smooth terminal experience
- 🚀 **Performance** - Efficient handling of multiple file changes

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

## Controls

- `q` or `Ctrl+C` - Quit the application

## How It Works

```
File Change → fsnotify → Event Queue → Debouncer → State Manager → Diff Engine → TUI
```

1. **File Watcher**: Monitors directory using platform-native APIs (inotify, FSEvents, etc.)
2. **Debouncer**: Batches rapid successive changes (100ms window)
3. **State Manager**: Tracks file contents for comparison
4. **Diff Engine**: Computes unified diffs between old and new states
5. **TUI Renderer**: Displays colorized diffs in real-time

## Development

See [DEVELOPMENT.md](DEVELOPMENT.md) for architecture details and development roadmap.

## License

MIT
