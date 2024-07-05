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
   -p                           Path to watch files from (default: ".")
   -x value                     Relative directories or files to exclude
   -go                          Reload only when .go file changes.
   -buildArgs value             Additional go build arguments
   -h                           show help
```

## Pass arguments to your program

You can use `--` to pass arguments to your program

```shell
pulse . -- -v abc foo bar
```