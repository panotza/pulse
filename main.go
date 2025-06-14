package main

import (
	"flag"
	"log"
)

type excludeFlag []string

func (f *excludeFlag) String() string { return "" }

func (f *excludeFlag) Set(v string) error {
	*f = append(*f, v)
	return nil
}

type watchDirFlag []string

func (f *watchDirFlag) String() string { return "" }

func (f *watchDirFlag) Set(v string) error {
	*f = append(*f, v)
	return nil
}

type buildArgFlag []string

func (f *buildArgFlag) String() string { return "" }

func (f *buildArgFlag) Set(v string) error {
	*f = append(*f, v)
	return nil
}

var (
	excludes    excludeFlag
	buildArgs   buildArgFlag
	watchDirs   watchDirFlag
	workingDir  = *flag.String("cwd", ".", "Working directory of the executable.")
	prebuildCmd = *flag.String("pbc", "", "Command to run before build.")
)

func main() {
	flag.Var(&excludes, "x", "Exclude a directory or a file. can be set multiple times with gitignore pattern.")
	flag.Var(&buildArgs, "buildArgs", "Additional go build arguments.")
	flag.Var(&watchDirs, "wd", "Watching directory.")
	flag.Parse()
	args := flag.Args()

	if err := run(args); err != nil {
		log.Fatal(err)
	}
}
