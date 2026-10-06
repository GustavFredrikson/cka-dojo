package ui

import (
	"strings"
	"testing"
)

func TestRenderMarkdownForPager(t *testing.T) {
	in := "# Services\n\nText\n\n```bash\nkubectl get svc\n```\n"
	plain := renderMarkdown(in, false)
	if strings.Contains(plain, "```") {
		t.Error("code fences leaked into rendered output")
	}
	if !strings.Contains(plain, "Services\n") {
		t.Errorf("heading was not rendered: %q", plain)
	}
	if !strings.Contains(plain, "    kubectl get svc") {
		t.Errorf("code was not indented: %q", plain)
	}

	ansi := renderMarkdown(in, true)
	if !strings.Contains(ansi, "\033[1mServices\033[0m") {
		t.Errorf("heading is not bold in pager output: %q", ansi)
	}
}

func TestRenderTableAlignsWideCharacters(t *testing.T) {
	got := renderTable([]string{"", "LEVEL", "EXERCISE"}, [][]string{
		{"✓", "0", "mental model"},
		{"🔒", "2", "services-build"},
		{"·", "1", "services-follow"},
	})
	want := "    LEVEL  EXERCISE\n" +
		"    -----  --------\n" +
		"✓   0      mental model\n" +
		"🔒  2      services-build\n" +
		"·   1      services-follow\n"
	if got != want {
		t.Errorf("table is misaligned:\n%s\nwant:\n%s", got, want)
	}
}

func TestRenderTableWithoutHeaders(t *testing.T) {
	got := renderTable(nil, [][]string{{"lab", "node-not-ready"}, {"checkpoint", "1/3"}})
	want := "lab         node-not-ready\ncheckpoint  1/3\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
