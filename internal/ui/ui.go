// Package ui is the engine's only writer to the terminal. Everything the
// learner sees goes through here, and everything -- verbose or not -- is
// mirrored to ~/.cka-dojo/logs/dojo.log so "it didn't work on my Mac" reports
// come with evidence.
package ui

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

var (
	mu      sync.Mutex
	verbose bool
	logFile io.WriteCloser
	color   = os.Getenv("NO_COLOR") == "" && isTTY(os.Stdout)
)

func isTTY(f *os.File) bool {
	fi, err := f.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}

// IsTerminal reports whether normal output is attached to an interactive
// terminal. Commands use it to avoid opening pagers in scripts and tests.
func IsTerminal() bool { return isTTY(os.Stdout) }

// SetVerbose turns on step-level output.
func SetVerbose(v bool) { verbose = v }

// Verbose reports whether verbose output is on.
func Verbose() bool { return verbose }

// SetLogFile starts mirroring output to w.
func SetLogFile(w io.WriteCloser) { logFile = w }

// CloseLog flushes the log file.
func CloseLog() {
	if logFile != nil {
		logFile.Close()
		logFile = nil
	}
}

func mirror(level, msg string) {
	if logFile == nil {
		return
	}
	fmt.Fprintf(logFile, "%s %-5s %s\n", time.Now().Format(time.RFC3339), level, msg)
}

func emit(w io.Writer, level, prefix, msg string) {
	mu.Lock()
	defer mu.Unlock()
	mirror(level, msg)
	fmt.Fprintf(w, "%s%s\n", prefix, msg)
}

func tint(c, s string) string {
	if !color {
		return s
	}
	return c + s + "\033[0m"
}

// Info prints a plain line.
func Info(format string, a ...any) { emit(os.Stdout, "info", "", fmt.Sprintf(format, a...)) }

// Step prints a progress line. Steps are always shown: provisioning takes
// minutes and a silent CLI looks hung.
func Step(format string, a ...any) {
	emit(os.Stdout, "step", tint("\033[36m", "-> "), fmt.Sprintf(format, a...))
}

// Detail prints only under --verbose, but is always logged.
func Detail(format string, a ...any) {
	msg := fmt.Sprintf(format, a...)
	if verbose {
		emit(os.Stdout, "debug", "   ", msg)
		return
	}
	mu.Lock()
	defer mu.Unlock()
	mirror("debug", msg)
}

// OK prints a success line.
func OK(format string, a ...any) {
	emit(os.Stdout, "ok", tint("\033[32m", "✓ "), fmt.Sprintf(format, a...))
}

// Fail prints a failure line.
func Fail(format string, a ...any) {
	emit(os.Stderr, "fail", tint("\033[31m", "✗ "), fmt.Sprintf(format, a...))
}

// Warn prints a warning line.
func Warn(format string, a ...any) {
	emit(os.Stderr, "warn", tint("\033[33m", "! "), fmt.Sprintf(format, a...))
}

// Blank prints an empty line.
func Blank() { emit(os.Stdout, "info", "", "") }

// Heading prints a section title.
func Heading(format string, a ...any) {
	msg := fmt.Sprintf(format, a...)
	emit(os.Stdout, "info", "", tint("\033[1m", msg))
}

// Table renders aligned columns.
func Table(headers []string, rows [][]string) {
	mu.Lock()
	defer mu.Unlock()
	fmt.Fprint(os.Stdout, renderTable(headers, rows))
	for _, r := range rows {
		mirror("table", strings.Join(r, " | "))
	}
}

// renderTable aligns columns by the width a terminal draws, not by rune
// count, which is what text/tabwriter uses: 🔒 is one rune but two columns,
// and counting it as one pushed every locked row in `dojo learn` one column
// to the right. Layout otherwise matches the tabwriter it replaces: two
// spaces between columns and no padding after a row's last cell.
func renderTable(headers []string, rows [][]string) string {
	all := rows
	if len(headers) > 0 {
		underline := make([]string, len(headers))
		for i, h := range headers {
			underline[i] = strings.Repeat("-", displayWidth(h))
		}
		all = append([][]string{headers, underline}, rows...)
	}

	var widths []int
	for _, r := range all {
		for i, cell := range r[:max(len(r)-1, 0)] {
			if i == len(widths) {
				widths = append(widths, 0)
			}
			widths[i] = max(widths[i], displayWidth(cell))
		}
	}

	var b strings.Builder
	for _, r := range all {
		for i, cell := range r {
			b.WriteString(cell)
			if i < len(r)-1 {
				b.WriteString(strings.Repeat(" ", widths[i]-displayWidth(cell)+2))
			}
		}
		b.WriteByte('\n')
	}
	return b.String()
}

// displayWidth is the number of terminal columns s occupies. It covers what
// dojo prints -- ASCII, the narrow status marks, emoji and CJK -- rather than
// the whole of Unicode East Asian Width.
func displayWidth(s string) int {
	w := 0
	for _, r := range s {
		if isWide(r) {
			w += 2
		} else {
			w++
		}
	}
	return w
}

func isWide(r rune) bool {
	switch {
	case r >= 0x1F300 && r <= 0x1FAFF: // pictographs and emoji, including 🔒
		return true
	case r >= 0x1100 && r <= 0x115F, // Hangul Jamo
		r >= 0x2E80 && r <= 0xA4CF, // CJK, kana, Yi
		r >= 0xAC00 && r <= 0xD7A3, // Hangul syllables
		r >= 0xF900 && r <= 0xFAFF, // CJK compatibility ideographs
		r >= 0xFF00 && r <= 0xFF60, // full-width forms
		r >= 0xFFE0 && r <= 0xFFE6,
		r >= 0x20000 && r <= 0x3FFFD:
		return true
	}
	return false
}

// Markdown renders a subset of Markdown for the terminal: headings are bolded,
// fenced code blocks are indented, everything else passes through. Lessons and
// tasks are short, so this stays deliberately dumb.
func Markdown(src string) {
	inCode := false
	for _, line := range strings.Split(strings.TrimRight(src, "\n"), "\n") {
		switch {
		case strings.HasPrefix(line, "```"):
			inCode = !inCode
			continue
		case inCode:
			emit(os.Stdout, "info", "", tint("\033[2m", "    "+line))
		case strings.HasPrefix(line, "#"):
			emit(os.Stdout, "info", "", tint("\033[1m", strings.TrimLeft(line, "# ")))
		default:
			emit(os.Stdout, "info", "", line)
		}
	}
}

// PageMarkdown renders Markdown through less when output is interactive and
// the lesson is long enough to need it. Redirected output always remains plain
// and deterministic, which keeps `dojo learn ... | cat` useful.
func PageMarkdown(src string, noPager bool) error {
	if noPager || !IsTerminal() {
		Markdown(src)
		return nil
	}
	less, err := exec.LookPath("less")
	if err != nil {
		Markdown(src)
		return nil
	}

	cmd := exec.Command(less, "-R", "-F", "-X")
	cmd.Stdin = strings.NewReader(renderMarkdown(src, os.Getenv("NO_COLOR") == ""))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// renderMarkdown is the string-producing counterpart to Markdown, used by a
// pager. It intentionally implements only the small subset our lessons use.
func renderMarkdown(src string, ansi bool) string {
	var out strings.Builder
	inCode := false
	for _, line := range strings.Split(strings.TrimRight(src, "\n"), "\n") {
		switch {
		case strings.HasPrefix(line, "```"):
			inCode = !inCode
			continue
		case inCode:
			line = "    " + line
			if ansi {
				line = "\033[2m" + line + "\033[0m"
			}
		case strings.HasPrefix(line, "#"):
			line = strings.TrimLeft(line, "# ")
			if ansi {
				line = "\033[1m" + line + "\033[0m"
			}
		}
		out.WriteString(line)
		out.WriteByte('\n')
	}
	return out.String()
}
