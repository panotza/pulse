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
pulse . # refer to current directory
pulse /path/to/your/project
```

Options

```txt
   -x value                     Relative directories or files to exclude.
   -go                          Reload only when .go file changes.
   -xp                          Disable built-in preset.
   -buildArgs value             Additional go build arguments.
   -h                           show help.
```

## Pass arguments to your program

You can use `--` to pass arguments to your program

```shell
pulse . -- -v abc foo bar
```

## Built-in exclude preset list

this is built-in exclude list enable by default (you can disable using `-xp` flag)

```
.git
.idea
.yarn
.vscode
.github
node_modules
```