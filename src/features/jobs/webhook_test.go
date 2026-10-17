package jobs

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// runRendered renders tmpl with name as the job name and runs it the way
// executeWebhookCommand does, returning stdout.
func runRendered(t *testing.T, tmpl, name string) string {
	t.Helper()
	command, env, err := renderWebhookCommand(tmpl, webhookValues{Name: name, Status: "completed"})
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

func TestWebhookCommandIsNotInjectable(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "pwned")
	payloads := []string{
		`x"; touch ` + marker + `; echo "`,
		`$(touch ` + marker + `)`,
		"`touch " + marker + "`",
		`'; touch ` + marker + `; echo '`,
	}
	for _, payload := range payloads {
		// The documented form: values inside double quotes.
		if got := runRendered(t, `printf '%s' "{{.Name}}"`, payload); got != payload {
			t.Errorf("payload %q: expected it echoed verbatim, got %q", payload, got)
		}
		// Unquoted placeholders must not execute anything either.
		runRendered(t, `echo {{.Name}} > /dev/null`, payload)
		if _, err := os.Stat(marker); err == nil {
			t.Fatalf("payload %q executed a command", payload)
		}
	}
}

func TestWebhookCommandRendersAllFields(t *testing.T) {
	command, env, err := renderWebhookCommand(`printf '%s|%s|%s|%s|%s' "{{.Name}}" "{{.Type}}" "{{.Status}}" "{{.Message}}" "{{.Duration}}"`,
		webhookValues{Name: "Download Album", Type: "download_album", Status: "completed", Message: "done", Duration: "3s"})
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("/bin/sh", "-c", command)
	cmd.Env = append(os.Environ(), env...)
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	if want := "Download Album|download_album|completed|done|3s"; string(out) != want {
		t.Errorf("expected %q, got %q", want, out)
	}
}

func TestWebhookCommandUnknownField(t *testing.T) {
	if _, _, err := renderWebhookCommand(`echo {{.Nmae}}`, webhookValues{}); err == nil {
		t.Error("expected an error for an unknown template field")
	}
}
