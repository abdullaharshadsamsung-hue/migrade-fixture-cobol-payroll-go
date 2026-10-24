package main

import (
	"io"
	"os"
	"testing"
)

func chdirTemp(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })
	return dir
}

func withStdio(t *testing.T, input string, f func()) string {
	t.Helper()
	inR, inW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(inW, input); err != nil {
		t.Fatal(err)
	}
	inW.Close()

	outR, outW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}

	oldIn, oldOut, oldErr := os.Stdin, os.Stdout, os.Stderr
	os.Stdin, os.Stdout, os.Stderr = inR, outW, outW

	done := make(chan []byte)
	go func() {
		b, _ := io.ReadAll(outR)
		done <- b
	}()

	func() {
		defer func() {
			os.Stdin, os.Stdout, os.Stderr = oldIn, oldOut, oldErr
		}()
		f()
	}()
	outW.Close()
	inR.Close()
	return string(<-done)
}

func TestRunDoesNotPanic(t *testing.T) {
	chdirTemp(t)
	var code int
	withStdio(t, "", func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("run panicked: %v", r)
			}
		}()
		code = run(nil)
	})
	if code < 0 {
		t.Errorf("run returned negative exit code %d", code)
	}
}

func TestRunDeterministic(t *testing.T) {
	chdirTemp(t)
	var code1, code2 int
	out1 := withStdio(t, "", func() { code1 = run(nil) })
	out2 := withStdio(t, "", func() { code2 = run(nil) })
	if code1 != code2 {
		t.Errorf("exit codes differ: %d vs %d", code1, code2)
	}
	if out1 != out2 {
		t.Errorf("outputs differ: %q vs %q", out1, out2)
	}
}