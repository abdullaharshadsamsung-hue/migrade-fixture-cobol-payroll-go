package main

import (
	"fmt"
	"os"
)

// run executes the program with its command-line arguments and returns
// its exit code.
func run(args []string) int {
	fmt.Fprintln(os.Stderr, "the migrated program is not wired into cmd/app/main.go yet")
	return 1
}

func main() {
	os.Exit(run(os.Args[1:]))
}
