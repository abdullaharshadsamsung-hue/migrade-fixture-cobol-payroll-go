package main

import (
	"bufio"
	"context"
	"os"
	"os/signal"

	"migrated-app/cmd/payroll"
)

// run executes the program with its command-line arguments and returns
// its exit code.
func run(args []string) int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	out := bufio.NewWriter(os.Stdout)
	code := payroll.Main(ctx, args, os.Stdin, out, os.Stderr)
	if err := out.Flush(); err != nil && code == 0 {
		code = 1
	}
	return code
}

func main() {
	os.Exit(run(os.Args[1:]))
}