package jobs

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/contre95/soulsolid/src/features/config"
	"github.com/contre95/soulsolid/src/music"
)

// runRendered renders tmpl with msg as the job message and runs it the way
// WebhookTask does, returning stdout.
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

// runIn renders tmpl with v and runs it in dir, returning stdout.
func runIn(t *testing.T, dir, tmpl string, v webhookValues) string {
	t.Helper()
	command, env, err := renderWebhookCommand(tmpl, v)
	if err != nil {
		t.Fatalf("render %q failed: %v", tmpl, err)
	}
	cmd := exec.Command("/bin/sh", "-c", command)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("command %q failed: %v", command, err)
	}
	return string(out)
}

func TestWebhookCommandIsNotInjectable(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "pwned")
	for _, payload := range []string{
		`x"; touch ` + marker + `; echo "`,
		`$(touch ` + marker + `)`,
		"`touch " + marker + "`",
		`'; touch ` + marker + `; echo '`,
	} {
		v := webhookValues{Name: payload}
		if got := runIn(t, dir, `printf '%s' "{{.Name}}"`, v); got != payload {
			t.Errorf("payload %q: expected it echoed verbatim, got %q", payload, got)
		}
		runIn(t, dir, `echo {{.Name}} > /dev/null`, v)
		if _, err := os.Stat(marker); err == nil {
			t.Fatalf("payload %q executed a command", payload)
		}
	}
}

// With curl -d {{.Message}}, a message like "x --data-binary @/etc/passwd" must
// not become extra curl options, and "*" must not glob.
func TestWebhookPlaceholderIsOneArgument(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "secret.txt"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{
		"x --data-binary @/etc/passwd",
		"*",
		"a  b\tc",
		"it's \"quoted\" $HOME `id` \\",
	} {
		want := "[" + value + "]\n"
		for _, tmpl := range []string{
			`printf '[%s]\n' {{.Message}}`,
			`printf '[%s]\n' "{{.Message}}"`,
			`printf '[%s]\n' '{{.Message}}'`,
		} {
			if got := runIn(t, dir, tmpl, webhookValues{Message: value}); got != want {
				t.Errorf("template %s with %q: expected one argument %q, got %q", tmpl, value, want, got)
			}
		}
	}

	got := runIn(t, dir, `printf '[%s]\n' prefix-{{.Message}}-suffix 'json:{"m":"{{.Message}}"}'`, webhookValues{Message: "a b"})
	if want := "[prefix-a b-suffix]\n[json:{\"m\":\"a b\"}]\n"; got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

func TestWebhookCommandRendersAllFields(t *testing.T) {
	got := runIn(t, t.TempDir(), `printf '%s|%s|%s|%s|%s' "{{.Name}}" "{{.Type}}" "{{.Status}}" "{{.Message}}" "{{.Duration}}"`,
		webhookValues{Name: "Download Album", Type: "download_album", Status: "completed", Message: "done", Duration: "3s"})
	if want := "Download Album|download_album|completed|done|3s"; got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

func TestWebhookCommandRejects(t *testing.T) {
	for _, tmpl := range []string{
		"echo {{.Nmae}}",
		"echo $(printf %s {{.Name}})",
		"echo \"$(printf %s {{.Name}})\"",
		"echo `printf %s {{.Name}}`",
		"cat <<EOF\n{{.Name}}\nEOF",
		"echo $'{{.Name}}'",
	} {
		if _, _, err := renderWebhookCommand(tmpl, webhookValues{}); err == nil {
			t.Errorf("expected template %q to be rejected", tmpl)
		}
	}
	if _, _, err := renderWebhookCommand(`echo \$\(x\) "\$(y)" {{.Name}}`, webhookValues{}); err != nil {
		t.Errorf("escaped $( should not be rejected: %v", err)
	}
}

// --- WebhookTask ---

func testJob(buf *bytes.Buffer) *music.Job {
	return &music.Job{Logger: slog.New(slog.NewTextHandler(buf, nil))}
}

func TestWebhookTaskTimeoutKillsProcessGroup(t *testing.T) {
	var logs bytes.Buffer
	task := &WebhookTask{timeout: 200 * time.Millisecond}
	start := time.Now()
	// The backgrounded sleep holds the output pipe open: only a group kill ends it.
	_, err := task.Execute(context.Background(), testJob(&logs), WebhookParams{Command: "sleep 5 & sleep 5"}, func(int, string) {})
	if err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("expected a timeout error, got %v", err)
	}
	if elapsed := time.Since(start); elapsed > 3*time.Second {
		t.Errorf("timeout took %s; the process group was not killed", elapsed)
	}
}

func TestWebhookTaskCancel(t *testing.T) {
	var logs bytes.Buffer
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(100*time.Millisecond, cancel)
	_, err := (&WebhookTask{timeout: 10 * time.Second}).Execute(ctx, testJob(&logs), WebhookParams{Command: "sleep 5"}, func(int, string) {})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestLineLoggerTruncates(t *testing.T) {
	var lines []string
	l := &lineLogger{log: func(msg string, _ ...any) { lines = append(lines, msg) }}
	chunk := strings.Repeat("x", 1023) + "\n"
	for range 2 * webhookOutputLimit / len(chunk) {
		l.Write([]byte(chunk))
	}
	l.flush()
	total := 0
	for _, line := range lines[:len(lines)-1] {
		total += len(line) + 1
	}
	if total > webhookOutputLimit {
		t.Errorf("logged %d bytes, over the %d limit", total, webhookOutputLimit)
	}
	if last := lines[len(lines)-1]; !strings.Contains(last, "truncated") {
		t.Errorf("expected a truncation notice, got %q", last)
	}
}

// --- Service ---

type testParams struct{}

// testTask blocks until release is closed (when set).
type testTask struct{ release chan struct{} }

func (t *testTask) Execute(ctx context.Context, _ *music.Job, _ testParams, _ func(int, string)) (map[string]any, error) {
	if t.release != nil {
		<-t.release
	}
	return nil, nil
}

func (t *testTask) Cleanup(*music.Job) error { return nil }

func newTestService(t *testing.T, trigger, command string) *Service {
	t.Helper()
	dir := t.TempDir()
	cfgYAML := "libraryPath: " + filepath.Join(dir, "lib") + "\n" +
		"downloadPath: " + filepath.Join(dir, "dl") + "\n" +
		"database:\n  path: " + filepath.Join(dir, "db") + "\n" +
		"jobs:\n  log: true\n  log_path: " + filepath.Join(dir, "logs") + "\n" +
		"  webhooks:\n    enabled: true\n    job_trigger: [" + trigger + "]\n    command: '" + strings.ReplaceAll(command, "'", "''") + "'\n"
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(cfgYAML), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.NewManager(path)
	if err != nil {
		t.Fatal(err)
	}
	return NewService(cfg)
}

// waitFor polls the service's jobs until cond holds, failing after 5s.
func waitFor(t *testing.T, s *Service, what string, cond func([]*music.Job) bool) []*music.Job {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		if jobs := s.GetJobs(); cond(jobs) {
			return jobs
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func ofType(jobs []*music.Job, jobType string) []*music.Job {
	var out []*music.Job
	for _, j := range jobs {
		if j.Type == jobType {
			out = append(out, j)
		}
	}
	return out
}

func TestWebhookJobLogsOutput(t *testing.T) {
	s := newTestService(t, `"*"`, `echo "done {{.Name}}"; echo oops >&2`)
	s.RegisterHandler("test_job", NewBaseTaskHandler[testParams](&testTask{}))
	if _, err := s.StartJob("test_job", "My Job", nil); err != nil {
		t.Fatal(err)
	}

	jobs := waitFor(t, s, "webhook job to complete", func(jobs []*music.Job) bool {
		w := ofType(jobs, webhookJobType)
		return len(w) == 1 && w[0].Status == music.JobStatusCompleted
	})
	webhook := ofType(jobs, webhookJobType)[0]
	if webhook.Name != "Webhook: My Job" {
		t.Errorf("unexpected webhook job name %q", webhook.Name)
	}
	logs, err := os.ReadFile(webhook.LogPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(logs), `msg="done My Job"`) || !strings.Contains(string(logs), "level=WARN msg=oops") {
		t.Errorf("expected stdout and stderr in the job log, got:\n%s", logs)
	}

	// A finished webhook job must not trigger another webhook.
	time.Sleep(300 * time.Millisecond)
	if n := len(ofType(s.GetJobs(), webhookJobType)); n != 1 {
		t.Errorf("expected exactly 1 webhook job, got %d", n)
	}
}

func TestWebhookJobFailure(t *testing.T) {
	s := newTestService(t, "test_job", "exit 7")
	s.RegisterHandler("test_job", NewBaseTaskHandler[testParams](&testTask{}))
	if _, err := s.StartJob("test_job", "My Job", nil); err != nil {
		t.Fatal(err)
	}
	jobs := waitFor(t, s, "webhook job to fail", func(jobs []*music.Job) bool {
		w := ofType(jobs, webhookJobType)
		return len(w) == 1 && w[0].Status == music.JobStatusFailed
	})
	if msg := ofType(jobs, webhookJobType)[0].Message; !strings.Contains(msg, "exit status 7") {
		t.Errorf("expected the exit status in the message, got %q", msg)
	}
}

// A running webhook job must neither wait for queued jobs nor hold them back.
func TestWebhookJobsDontBlockQueue(t *testing.T) {
	s := newTestService(t, "block", "sleep 2")
	release := make(chan struct{})
	s.RegisterHandler("block", NewBaseTaskHandler[testParams](&testTask{release: release}))
	s.RegisterHandler("test_job", NewBaseTaskHandler[testParams](&testTask{}))

	a, _ := s.StartJob("block", "A", nil)
	b, _ := s.StartJob("test_job", "B", nil)
	if job, _ := s.GetJob(b); job.Status != music.JobStatusPending {
		t.Fatalf("expected B to queue behind A, got %s", job.Status)
	}
	close(release)

	waitFor(t, s, "B to complete while A's webhook runs", func(jobs []*music.Job) bool {
		job, _ := s.GetJob(b)
		w := ofType(jobs, webhookJobType)
		return job.Status == music.JobStatusCompleted && len(w) == 1 && w[0].Status == music.JobStatusRunning
	})
	if job, _ := s.GetJob(a); job.Status != music.JobStatusCompleted {
		t.Errorf("expected A completed, got %s", job.Status)
	}
}
