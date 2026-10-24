// Command payroll runs the Go port of the COBOL PAYROLL batch program.
package main

import (
	"bufio"
	"context"
	"os"
	"os/signal"
	"syscall"

	"migrated-app/internal/payroll"
)

func run(args []string) int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
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
