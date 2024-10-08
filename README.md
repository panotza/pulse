# Pulse

Pulse is a command-line utility designed for live-reloading Go applications, featuring intelligent file
change detection and optimization for a seamless, fast development feedback loop.

## Installation

```shell
go install github.com/panotza/pulse@main
```

## Basic usage

in your root Go project run

```shell
pulse
```

or

```shell
pulse . # watch current directory and also go run current directory package
pulse /path/to/your/package (directory or go file) # watch current directory and go run /path/to/your/package
pulse -wd ./sub/folder . # watch only specific folder
```

Options

```txt
   -wd                          Watching Directory.
   -cwd                         Working directory of the executable.
   -x value                     Relative directories or files to exclude.
   -go                          Reload only when .go files change.
   -xp                          Disable the built-in preset.
   -buildArgs value             Additional Go build arguments.
   -h                           Show help.
```

## Pass arguments to your program

You can use `--` to pass arguments to your program

```shell
pulse . -- -v abc foo bar
```

## Built-in exclude preset list

this is a built-in exclude list enabled by default (you can disable it using the `-xp` flag)

```
.git
.idea
.yarn
.vscode
.github
node_modules
```