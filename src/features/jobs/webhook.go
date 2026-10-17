package jobs

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"text/template"
	"time"

	"github.com/contre95/soulsolid/src/music"
)

const (
	webhookJobType = "webhook"
	webhookTimeout = 30 * time.Second
	// webhookOutputLimit caps how much of each output stream is logged.
	webhookOutputLimit = 64 << 10
)

// WebhookParams is the metadata of a webhook job: the rendered command and the
// SOULSOLID_JOB_* values it reads.
type WebhookParams struct {
	Command string   `json:"command" validate:"required"`
	Env     []string `json:"env"`
}

// WebhookTask runs a webhook command, streaming its output into the job log.
type WebhookTask struct {
	timeout time.Duration
}

func (t *WebhookTask) Execute(ctx context.Context, job *music.Job, params WebhookParams, progressUpdater func(int, string)) (map[string]any, error) {
	runCtx, cancel := context.WithTimeout(ctx, t.timeout)
	defer cancel()

	cmd := exec.CommandContext(runCtx, "/bin/sh", "-c", params.Command)
	cmd.Env = webhookEnviron(os.Environ(), params.Env)
	// Put the shell and all its children in a new process group so cancel and
	// timeout kill the entire tree and no grandchildren are orphaned to PID 1.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
	// Don't wait forever on pipes a background grandchild keeps open.
	cmd.WaitDelay = 2 * time.Second

	stdout := &lineLogger{log: job.Logger.Info}
	stderr := &lineLogger{log: job.Logger.Warn}
	cmd.Stdout, cmd.Stderr = stdout, stderr

	job.Logger.Info("Running webhook", "command", params.Command)
	progressUpdater(10, "Running webhook command")
	err := cmd.Run()
	stdout.flush()
	stderr.flush()

	switch {
	case ctx.Err() != nil:
		return nil, ctx.Err()
	case errors.Is(runCtx.Err(), context.DeadlineExceeded):
		return nil, fmt.Errorf("webhook command timed out after %s", t.timeout)
	case err != nil:
		return nil, fmt.Errorf("webhook command failed: %w", err)
	}
	return nil, nil
}

func (t *WebhookTask) Cleanup(job *music.Job) error { return nil }

// webhookEnvPrefix marks the server environment variables a webhook may read.
const webhookEnvPrefix = "SOULSOLID_WEBHOOK_"

// webhookEnviron builds a webhook's environment: PATH and SOULSOLID_WEBHOOK_*
// from the server, so other secrets the server holds never reach the command,
// plus the job's SOULSOLID_JOB_* values.
func webhookEnviron(environ, jobEnv []string) []string {
	env := make([]string, 0, len(jobEnv)+1)
	for _, kv := range environ {
		if strings.HasPrefix(kv, "PATH=") || strings.HasPrefix(kv, webhookEnvPrefix) {
			env = append(env, kv)
		}
	}
	return append(env, jobEnv...)
}

// lineLogger logs each line written to it, up to webhookOutputLimit bytes.
type lineLogger struct {
	log       func(msg string, args ...any)
	mu        sync.Mutex
	buf       []byte
	written   int
	truncated bool
}

func (l *lineLogger) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	n := len(p)
	if l.truncated {
		return n, nil
	}
	if room := webhookOutputLimit - l.written; len(p) > room {
		p = p[:room]
		l.truncated = true
	}
	l.written += len(p)
	l.buf = append(l.buf, p...)
	for {
		i := bytes.IndexByte(l.buf, '\n')
		if i < 0 {
			break
		}
		l.log(strings.TrimSuffix(string(l.buf[:i]), "\r"))
		l.buf = l.buf[i+1:]
	}
	if l.truncated {
		l.flushLocked()
		l.log(fmt.Sprintf("[output truncated after %d bytes]", webhookOutputLimit))
	}
	return n, nil
}

func (l *lineLogger) flush() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.flushLocked()
}

func (l *lineLogger) flushLocked() {
	if len(l.buf) > 0 {
		l.log(string(l.buf))
		l.buf = nil
	}
}

// webhookValues are the job details a webhook command template can reference.
type webhookValues struct {
	Name     string
	Type     string
	Status   string
	Message  string
	Duration string
}

// webhookEnv maps each template field to the environment variable carrying it.
var webhookEnv = []struct{ field, env string }{
	{"Name", "SOULSOLID_JOB_NAME"},
	{"Type", "SOULSOLID_JOB_TYPE"},
	{"Status", "SOULSOLID_JOB_STATUS"},
	{"Message", "SOULSOLID_JOB_MESSAGE"},
	{"Duration", "SOULSOLID_JOB_DURATION"},
}

// renderWebhookCommand renders the template with job values passed as env vars,
// never spliced into the shell source, to prevent command injection.
func renderWebhookCommand(tmplText string, v webhookValues) (string, []string, error) {
	tmpl, err := template.New("webhook").Option("missingkey=error").Parse(tmplText)
	if err != nil {
		return "", nil, fmt.Errorf("parse webhook template: %w", err)
	}

	values := map[string]string{
		"Name":     v.Name,
		"Type":     v.Type,
		"Status":   v.Status,
		"Message":  v.Message,
		"Duration": v.Duration,
	}
	// NUL-delimited markers, resolved by quoteWebhookPlaceholders.
	markers := make(map[string]string, len(webhookEnv))
	env := make([]string, 0, len(webhookEnv))
	for _, e := range webhookEnv {
		markers[e.field] = "\x00" + e.env + "\x00"
		env = append(env, e.env+"="+values[e.field])
	}

	var rendered strings.Builder
	if err := tmpl.Execute(&rendered, markers); err != nil {
		return "", nil, fmt.Errorf("execute webhook template: %w", err)
	}
	command, err := quoteWebhookPlaceholders(rendered.String())
	if err != nil {
		return "", nil, err
	}
	return command, env, nil
}

func isWebhookEnv(name string) bool {
	for _, e := range webhookEnv {
		if e.env == name {
			return true
		}
	}
	return false
}

// quoteWebhookPlaceholders replaces each marker with an env var reference quoted
// for its context, so it expands to one literal argument. Contexts it can't track
// ($(...), backticks, heredocs, $'...') are rejected.
func quoteWebhookPlaceholders(cmd string) (string, error) {
	const (
		unquoted = iota
		single
		double
		comment
	)
	var out strings.Builder
	state := unquoted
	opaque := ""  // first construct seen whose inner quoting isn't tracked
	escaped := -1 // index of the last backslash-escaped byte
	for i := 0; i < len(cmd); i++ {
		c := cmd[i]
		if c == 0 {
			end := strings.IndexByte(cmd[i+1:], 0)
			if end < 0 || !isWebhookEnv(cmd[i+1:i+1+end]) {
				return "", fmt.Errorf("malformed webhook placeholder: template functions must not modify job fields")
			}
			ref := "${" + cmd[i+1:i+1+end] + "}"
			i += end + 1
			if opaque != "" {
				return "", fmt.Errorf("webhook placeholders can't be used after %s; read the SOULSOLID_JOB_* environment variables there instead", opaque)
			}
			switch state {
			case unquoted:
				out.WriteString(`"` + ref + `"`)
			case double:
				out.WriteString(ref)
			case single:
				out.WriteString(`'"` + ref + `"'`)
			case comment:
				out.WriteString(ref) // ignored by the shell
			}
			continue
		}
		out.WriteByte(c)
		switch state {
		case comment:
			if c == '\n' {
				state = unquoted
			}
			continue
		case single:
			if c == '\'' {
				state = unquoted
			}
			continue
		case unquoted, double:
			if c == '\\' && i+1 < len(cmd) && cmd[i+1] != 0 {
				i++
				out.WriteByte(cmd[i])
				escaped = i
				continue
			}
			rest := cmd[i:]
			switch {
			case c == '`':
				opaque = "a backtick"
			case strings.HasPrefix(rest, "$("):
				opaque = "a $( command substitution"
			case state == unquoted && strings.HasPrefix(rest, "$'"):
				opaque = "a $'...' string"
			case state == unquoted && strings.HasPrefix(rest, "<<"):
				opaque = "a heredoc"
			}
		}
		switch {
		case state == unquoted && c == '#' && (i == 0 || escaped != i-1 && strings.IndexByte(" \t\n;&|()<>", cmd[i-1]) >= 0):
			state = comment
		case state == unquoted && c == '\'':
			state = single
		case state == unquoted && c == '"':
			state = double
		case state == double && c == '"':
			state = unquoted
		}
	}
	return out.String(), nil
}
