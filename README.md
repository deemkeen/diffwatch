# diffwatch

A real-time CLI tool for watching file changes and visualizing diffs in your terminal.

![DiffWatch Demo](demo.gif)

## Features

- Real-time file watching with platform-native notifications
- Live diff visualization with colored output
- Recursive subdirectory watching
- Smart file filtering (ignores shell history, lock files, temp files)
- Event coalescing to handle rapid file changes
- 1MB file size limit for graceful handling of large files
- Beautiful TUI built with Bubbletea
- Binary file detection
- Automatic permission error handling

## Installation

### Homebrew (macOS/Linux)

```bash
brew install deemkeen/tap/diffwatch
```

### Go Install

```bash
go install github.com/deemkeen/diffwatch/cmd/diffwatch@latest
```

### Download Binary

Download the pre-built binary for your platform from the [latest release](https://github.com/deemkeen/diffwatch/releases/latest).

### Build from Source

```bash
git clone https://github.com/deemkeen/diffwatch.git
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
diffwatch -p /path/to/directory
# or
diffwatch -path /path/to/directory
```

Watch all subdirectories recursively:
```bash
diffwatch -p /path/to/directory -r
# or
diffwatch -path /path/to/directory -recursive
```

## Options

- `-p`, `-path` - Path to watch for changes (default: current directory)
- `-r`, `-recursive` - Watch all subdirectories recursively (default: false)
- `-h` - Show help

## Controls

- `q` or `Ctrl+C` - Quit the application

## How It Works

```
File Change → fsnotify → Debouncer → State Manager → Diff Engine → TUI
```

1. **File Watcher** - Monitors files using platform-native APIs (inotify, FSEvents, etc.)
2. **Event Coalescing** - Batches rapid changes within a 200ms window to prevent TUI flicker
3. **File Filtering** - Automatically ignores noisy files (shell history, lock files, temp files)
4. **State Manager** - Tracks file contents for comparison
5. **Diff Engine** - Computes unified diffs between versions
6. **TUI Renderer** - Displays colorized diffs in real-time with file status

## What's Filtered Out

DiffWatch automatically ignores common noisy files:
- Shell history files (`.zsh_history`, `.bash_history`, etc.)
- Lock files (`*.lock`, `*.LOCK`)
- Vim/Emacs temporary files (`.*.swp`, `#*#`, `*~`)
- Common dotfiles (`.lesshst`, `.viminfo`, `.recently-used`)
- Build directories (`.git`, `node_modules`, `.cache`, etc.)

## License

MIT
