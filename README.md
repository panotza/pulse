# Pulse

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/License-BSD%203--Clause-blue.svg)](LICENSE)

Pulse is a powerful command-line utility designed for live-reloading Go applications during development. It features intelligent file change detection and optimization for a seamless, fast development feedback loop.

## Features

- 🚀 **Fast live-reloading** - Automatically rebuilds and restarts your Go application when files change
- 🎯 **Intelligent file watching** - Monitors Go source files with gitignore pattern filtering
- 🚫 **Smart ignore patterns** - Respects `.gitignore`, `.pulseignore`, and command-line exclusions
- 📁 **Flexible directory watching** - Watch specific directories or exclude unwanted paths
- 🔧 **Customizable build process** - Support for custom build arguments and pre-build commands
- 📋 **Argument forwarding** - Pass arguments directly to your application

## Prerequisites

- Go 1.24 or higher

## Installation

Install Pulse using Go's package manager:

```shell
go install github.com/panotza/pulse@main
```

## Quick Start

In your Go project root directory, simply run:

```shell
pulse
```

This will:
1. Watch the current directory for file changes
2. Automatically rebuild your application when changes are detected
3. Restart the application with the new build

## Usage Examples

### Basic Usage

```shell
# Watch current directory and run the current package
pulse

# Watch current directory and run a specific package
pulse .

# Watch current directory and run a specific package or file
pulse /path/to/your/package
```

### Advanced Usage

```shell
# Watch only specific directories
pulse -wd ./cmd -wd ./internal .

# Exclude specific directories or files (supports gitignore patterns)
pulse -x ./vendor -x ./tmp -x "*.log" -x "test_*" .

# Use custom build arguments
pulse -buildArgs="-tags=dev -ldflags=-X main.version=dev" .

# Run a command before each build
pulse -pbc="go generate ./..." .

# Set working directory for the executable
pulse -cwd=/path/to/runtime/dir .
```

## Command Line Options

| Flag | Description | Example |
|------|-------------|---------|
| `-wd` | Specify directories to watch for changes | `-wd ./cmd -wd ./internal` |
| `-cwd` | Set working directory for the executable | `-cwd ./build` |
| `-x` | Exclude directories or files from watching (supports gitignore patterns) | `-x ./vendor -x "*.log"` |
| `-buildArgs` | Additional arguments passed to `go build` | `-buildArgs="-tags=dev"` |
| `-pbc` | Command to run before each build | `-pbc="go generate"` |
| `-h` | Show help information | `-h` |

## Passing Arguments to Your Application

Use `--` to separate Pulse arguments from your application arguments:

```shell
# Pass flags and arguments to your application
pulse . -- -v -port=8080 --config=dev.json
```

## How It Works

Pulse monitors your Go source files using an efficient file system watcher. When changes are detected:

1. **Pre-build commands** - Optional commands (like `go generate`) are executed first
2. **Building** - Your application is compiled with `go build`
3. **Process management** - The old process is terminated and the new one is started
4. **Output streaming** - Your application's output is displayed in real-time

### File Exclusion System

Pulse uses a layered approach to determine which files to watch, applying ignore patterns in the following order:

1. **`.gitignore`** - Standard Git ignore patterns from your repository
2. **`.pulseignore`** - Pulse-specific ignore patterns (same syntax as `.gitignore`)
3. **`-x` flags** - Command-line exclusion patterns

**Important:** Later patterns can override earlier ones, just like Git's ignore system. This means:
- `.pulseignore` patterns can override `.gitignore` patterns
- Command-line `-x` flags have the highest priority and can override both files

#### Example `.pulseignore` file:
```gitignore
# Pulse-specific ignores
*.tmp
debug/
!important.log
build/*.temp
```

## Troubleshooting

### Getting Help

- Run `pulse -h` for command-line help
- Check the [Issues](https://github.com/panotza/pulse/issues) page for known problems
- Create a new issue if you encounter a bug

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request. For major changes, please open an issue first to discuss what you would like to change.

## License

This project is licensed under the BSD 3-Clause License - see the [LICENSE](LICENSE) file for details.