// Package payroll is the Go port of the COBOL PAYROLL batch program.
//
// It reads fixed-width employee records from employees.txt, computes each
// employee's pay (time-and-a-half for hours over 40), writes one "ID PAY"
// line per record to payroll.txt and prints "TOTAL <sum>" to standard output.
//
// All arithmetic is done in exact int64 cents to reproduce COBOL fixed-point
// semantics (PIC 9(n)V99, COMPUTE ROUNDED, high-order truncation).
package payroll

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

const (
	inputPath  = "employees.txt"
	outputPath = "payroll.txt"

	// recordLen is the length of EMP-REC: 9(4) X 9(3) X 9(5).
	recordLen = 14

	// regularHours is the threshold above which overtime applies.
	regularHours = 40

	// payModulus emulates high-order truncation into WS-PAY PIC 9(7)V99.
	payModulus = 1_000_000_000
	// totalModulus emulates high-order truncation into WS-TOTAL PIC 9(9)V99
	// (ADD without ON SIZE ERROR).
	totalModulus = 100_000_000_000
)

// Record is one parsed EMP-REC.
type Record struct {
	// ID is the raw 4-byte EMP-ID field, emitted verbatim on output.
	ID string
	// Hours is EMP-HOURS (0-999).
	Hours int64
	// RateCents is EMP-RATE, the hourly rate in cents (0-99999).
	RateCents int64
}

// ParseRecord decodes one fixed-width line. Short lines are padded with
// spaces to 14 bytes and bytes beyond column 14 are ignored, mirroring
// LINE SEQUENTIAL reads. Digits are decoded as byte&0x0F, which is how
// GnuCOBOL treats unsigned DISPLAY numerics (a space decodes as 0).
func ParseRecord(line string) Record {
	line = strings.TrimRight(line, "\r\n")
	if len(line) < recordLen {
		line += strings.Repeat(" ", recordLen-len(line))
	}
	// MIGRATION_NOTE: non-numeric bytes in numeric fields are not rejected,
	// matching COBOL's unchecked DISPLAY numeric decoding.
	return Record{
		ID:        line[0:4],
		Hours:     decodeDigits(line[5:8]),
		RateCents: decodeDigits(line[9:14]),
	}
}

func decodeDigits(s string) int64 {
	var n int64
	for i := 0; i < len(s); i++ {
		n = n*10 + int64(s[i]&0x0F)
	}
	return n
}

// PayCents computes the pay for one record in cents. Hours over 40 are
// paid at 1.5x; the result is rounded half-up (COMPUTE ROUNDED) and
// truncated to fit PIC 9(7)V99.
func PayCents(r Record) int64 {
	var pay int64
	if r.Hours > regularHours {
		overtime := r.Hours - regularHours
		// Work in half-cents so 1.5x is exact: 2*(40*rate) + 3*ot*rate.
		halfCents := 2*regularHours*r.RateCents + 3*overtime*r.RateCents
		pay = (halfCents + 1) / 2
	} else {
		pay = r.Hours * r.RateCents
	}
	return pay % payModulus
}

// formatEdited renders cents using an edited picture Z(n)9.99, where
// intWidth is the total number of integer positions (n+1).
func formatEdited(cents int64, intWidth int) string {
	return fmt.Sprintf("%*d.%02d", intWidth, cents/100, cents%100)
}

// FormatPay renders cents as PIC Z(6)9.99 (10 characters).
func FormatPay(cents int64) string { return formatEdited(cents, 7) }

// FormatTotal renders cents as PIC Z(8)9.99 (12 characters).
func FormatTotal(cents int64) string { return formatEdited(cents, 9) }

// PayLine builds the output record: ID, a space, and the edited pay.
func PayLine(r Record, payCents int64) string {
	return r.ID + " " + FormatPay(payCents)
}

// Run processes every record from in, writes pay lines to out, and returns
// the grand total in cents. A trailing line without a newline counts as a
// record; an empty input yields no records.
func Run(ctx context.Context, in io.Reader, out io.Writer) (int64, error) {
	br := bufio.NewReader(in)
	bw := bufio.NewWriter(out)
	var total int64
	for {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		line, err := br.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return 0, fmt.Errorf("read %s: %w", inputPath, err)
		}
		atEOF := errors.Is(err, io.EOF)
		if line == "" && atEOF {
			break
		}
		rec := ParseRecord(line)
		pay := PayCents(rec)
		total = (total + pay) % totalModulus
		if _, werr := bw.WriteString(PayLine(rec, pay) + "\n"); werr != nil {
			return 0, fmt.Errorf("write %s: %w", outputPath, werr)
		}
		if atEOF {
			break
		}
	}
	if err := bw.Flush(); err != nil {
		return 0, fmt.Errorf("write %s: %w", outputPath, err)
	}
	return total, nil
}

func runMain(ctx context.Context, stdout io.Writer) error {
	// Open input before creating output, as OPEN INPUT ... OUTPUT does.
	in, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("open %s: %w", inputPath, err)
	}
	defer in.Close()

	out, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create %s: %w", outputPath, err)
	}

	total, err := Run(ctx, in, out)
	if err != nil {
		_ = out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("close %s: %w", outputPath, err)
	}
	fmt.Fprintln(stdout, "TOTAL "+FormatTotal(total))
	return nil
}

// Main runs the payroll program and returns its exit code.
func Main(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	_ = args
	_ = stdin
	if err := runMain(ctx, stdout); err != nil {
		fmt.Fprintln(stderr, "payroll:", err)
		return 1
	}
	return 0
}