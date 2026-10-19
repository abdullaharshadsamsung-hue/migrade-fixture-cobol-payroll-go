package payroll

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseRecord(t *testing.T) {
	tests := []struct {
		name string
		line string
		want Record
	}{
		{"normal", "0001 040 02500", Record{ID: "0001", Hours: 40, RateCents: 2500}},
		{"crlf", "0002 045 02000\r\n", Record{ID: "0002", Hours: 45, RateCents: 2000}},
		{"short padded", "0003 041", Record{ID: "0003", Hours: 41, RateCents: 0}},
		{"extra ignored", "0004 001 00100XYZ", Record{ID: "0004", Hours: 1, RateCents: 100}},
		{"leading zero id", "0000 000 00000", Record{ID: "0000", Hours: 0, RateCents: 0}},
		{"max", "9999 999 99999", Record{ID: "9999", Hours: 999, RateCents: 99999}},
		{"empty", "", Record{ID: "    ", Hours: 0, RateCents: 0}},
		{"spaces as zero", "0005  4  25 0", Record{ID: "0005", Hours: 40, RateCents: 2500}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseRecord(tt.line); got != tt.want {
				t.Errorf("ParseRecord(%q) = %+v, want %+v", tt.line, got, tt.want)
			}
		})
	}
}

func TestParseRecordNonNumericDoesNotPanic(t *testing.T) {
	r := ParseRecord("ABCD xyz qwert")
	if r.ID != "ABCD" {
		t.Errorf("ID = %q", r.ID)
	}
	if r.Hours < 0 || r.RateCents < 0 {
		t.Errorf("negative decode: %+v", r)
	}
}

func TestPayCents(t *testing.T) {
	tests := []struct {
		name  string
		hours int64
		rate  int64
		want  int64
	}{
		{"regular 40h", 40, 2500, 100000},
		{"overtime 45h", 45, 2000, 95000},
		{"half cent rounds up", 41, 1, 42},
		{"zero hours", 0, 2500, 0},
		{"zero rate", 50, 0, 0},
		{"under 40", 10, 1234, 12340},
		{"max no overflow", 999, 99999, 147848522},
		{"odd overtime exact", 42, 1, 43},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PayCents(Record{ID: "0001", Hours: tt.hours, RateCents: tt.rate})
			if got != tt.want {
				t.Errorf("PayCents(%d,%d) = %d, want %d", tt.hours, tt.rate, got, tt.want)
			}
			if got < 0 || got > 999999999 {
				t.Errorf("out of range: %d", got)
			}
		})
	}
}

func TestFormatPayAndTotal(t *testing.T) {
	pays := []struct {
		cents int64
		want  string
	}{
		{100000, "   1000.00"},
		{95000, "    950.00"},
		{42, "      0.42"},
		{0, "      0.00"},
		{999999999, "9999999.99"},
	}
	for _, tt := range pays {
		got := FormatPay(tt.cents)
		if got != tt.want || len(got) != 10 {
			t.Errorf("FormatPay(%d) = %q, want %q", tt.cents, got, tt.want)
		}
	}
	totals := []struct {
		cents int64
		want  string
	}{
		{0, "         0.00"[1:]},
		{195042, "      1950.42"[1:]},
		{99999999999, "999999999.99"},
	}
	for _, tt := range totals {
		got := FormatTotal(tt.cents)
		if got != tt.want || len(got) != 12 {
			t.Errorf("FormatTotal(%d) = %q, want %q", tt.cents, got, tt.want)
		}
	}
}

func TestPayLine(t *testing.T) {
	tests := []struct {
		rec  Record
		pay  int64
		want string
	}{
		{Record{ID: "0001"}, 100000, "0001    1000.00"},
		{Record{ID: "0002"}, 95000, "0002     950.00"},
		{Record{ID: "0003"}, 42, "0003       0.42"},
		{Record{ID: "0004"}, 0, "0004       0.00"},
	}
	for _, tt := range tests {
		if got := PayLine(tt.rec, tt.pay); got != tt.want {
			t.Errorf("PayLine = %q, want %q", got, tt.want)
		}
	}
}

func TestRun(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantOut   string
		wantTotal int64
	}{
		{"empty", "", "", 0},
		{"single regular", "0001 040 02500\n", "0001    1000.00\n", 100000},
		{"single overtime", "0002 045 02000\n", "0002     950.00\n", 95000},
		{"half cent", "0003 041 00001\n", "0003       0.42\n", 42},
		{"zero", "0004 000 02500\n0005 040 00000\n", "0004       0.00\n0005       0.00\n", 0},
		{
			"multiple in order no trailing newline",
			"0001 040 02500\n0002 045 02000\n0003 041 00001",
			"0001    1000.00\n0002     950.00\n0003       0.42\n",
			195042,
		},
		{"crlf", "0001 040 02500\r\n", "0001    1000.00\n", 100000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			total, err := Run(context.Background(), strings.NewReader(tt.input), &out)
			if err != nil {
				t.Fatalf("Run error: %v", err)
			}
			if out.String() != tt.wantOut {
				t.Errorf("out = %q, want %q", out.String(), tt.wantOut)
			}
			if total != tt.wantTotal {
				t.Errorf("total = %d, want %d", total, tt.wantTotal)
			}
			inLines := 0
			if tt.input != "" {
				inLines = len(strings.Split(strings.TrimSuffix(tt.input, "\n"), "\n"))
			}
			if got := strings.Count(out.String(), "\n"); got != inLines {
				t.Errorf("line count %d, want %d", got, inLines)
			}
		})
	}
}

type errReader struct{ err error }

func (r errReader) Read([]byte) (int, error) { return 0, r.err }

type errWriter struct{ err error }

func (w errWriter) Write([]byte) (int, error) { return 0, w.err }

func TestRunErrors(t *testing.T) {
	boom := errors.New("boom")
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	tests := []struct {
		name    string
		ctx     context.Context
		in      io.Reader
		out     io.Writer
		wantErr error
	}{
		{"read error", context.Background(), errReader{boom}, &bytes.Buffer{}, boom},
		{"write error", context.Background(), strings.NewReader("0001 040 02500\n"), errWriter{boom}, boom},
		{"canceled", canceled, strings.NewReader("0001 040 02500\n"), &bytes.Buffer{}, context.Canceled},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			total, err := Run(tt.ctx, tt.in, tt.out)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("err = %v, want %v", err, tt.wantErr)
			}
			if total != 0 {
				t.Errorf("total = %d, want 0", total)
			}
		})
	}
}

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

func captureStdout(t *testing.T, f func() error) (string, error) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	os.Stdout = w
	ferr := f()
	os.Stdout = old
	w.Close()
	b, _ := io.ReadAll(r)
	return string(b), ferr
}

func TestRunMain(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		existing   string
		wantFile   string
		wantStdout string
	}{
		{"empty", "", "", "", "TOTAL          0.00\n"},
		{
			"multiple overwrites",
			"0001 040 02500\n0002 045 02000\n0003 041 00001\n",
			"stale content\n",
			"0001    1000.00\n0002     950.00\n0003       0.42\n",
			"TOTAL       1950.42\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := chdirTemp(t)
			if err := os.WriteFile(filepath.Join(dir, inputPath), []byte(tt.input), 0o644); err != nil {
				t.Fatal(err)
			}
			if tt.existing != "" {
				if err := os.WriteFile(filepath.Join(dir, outputPath), []byte(tt.existing), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			stdout, err := captureStdout(t, func() error { return runMain(context.Background(), os.Stdout) })
			if err != nil {
				t.Fatalf("runMain: %v", err)
			}
			if stdout != tt.wantStdout {
				t.Errorf("stdout = %q, want %q", stdout, tt.wantStdout)
			}
			got, err := os.ReadFile(filepath.Join(dir, outputPath))
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tt.wantFile {
				t.Errorf("file = %q, want %q", got, tt.wantFile)
			}
		})
	}
}

func TestRunMainMissingInput(t *testing.T) {
	dir := chdirTemp(t)
	err := runMain(context.Background(), io.Discard)
	if err == nil {
		t.Fatal("expected error for missing input")
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("err = %v, want not-exist", err)
	}
	if _, statErr := os.Stat(filepath.Join(dir, outputPath)); !os.IsNotExist(statErr) {
		t.Errorf("output should not be created when input is missing")
	}
}