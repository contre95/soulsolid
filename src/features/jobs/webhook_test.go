package jobs

import (
	"os"
	"os/exec"
	"testing"
)

// runRendered renders tmpl with msg as the job message and runs it the way
// executeWebhookCommand does, returning stdout.
func runRendered(t *testing.T, tmpl, msg string) string {
	t.Helper()
	command, env, err := renderWebhookCommand(tmpl, webhookValues{Message: msg})
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	cmd := exec.Command("/bin/sh", "-c", command)
	cmd.Env = append(os.Environ(), env...)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("command %q failed: %v", command, err)
	}
	return string(out)
}

func TestWebhookQuotesInCommentsDoNotChangeQuoting(t *testing.T) {
	msg := "x --data-binary @/etc/passwd"
	for _, tmpl := range []string{
		"# notify \"ops\nprintf '%s|' {{.Message}}",
		"# a \"\nprintf '%s|' \"{{.Message}}\"",
		"# don't\nprintf '%s|' {{.Message}}",
		"true # it's {{.Message}}\nprintf '%s|' {{.Message}}",
		"printf '%s|' a\\ #b {{.Message}}",
	} {
		got := runRendered(t, tmpl, msg)
		if got != msg+"|" && got != "a #b|"+msg+"|" {
			t.Errorf("template %q: expected the message as one argument, got %q", tmpl, got)
		}
	}
}

func TestWebhookMalformedPlaceholder(t *testing.T) {
	for _, tmpl := range []string{
		`echo {{printf "%.10s" .Message}}`,
		"echo \x00 {{.Message}}",
	} {
		if _, _, err := renderWebhookCommand(tmpl, webhookValues{Message: "hi"}); err == nil {
			t.Errorf("template %q: expected an error", tmpl)
		}
	}
}
