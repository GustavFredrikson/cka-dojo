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
