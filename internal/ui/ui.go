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
	"text/tabwriter"
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
	tw := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	if len(headers) > 0 {
		fmt.Fprintln(tw, strings.Join(headers, "\t"))
		underline := make([]string, len(headers))
		for i, h := range headers {
			underline[i] = strings.Repeat("-", len(h))
		}
		fmt.Fprintln(tw, strings.Join(underline, "\t"))
	}
	for _, r := range rows {
		fmt.Fprintln(tw, strings.Join(r, "\t"))
		mirror("table", strings.Join(r, " | "))
	}
	tw.Flush()
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
