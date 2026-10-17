package mcp

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/contre95/soulsolid/src/features/config"
)

func TestRedactConfig(t *testing.T) {
	secret := "provider-secret"
	cfg := &config.Config{
		Telegram: config.Telegram{Token: "telegram-token"},
		Jobs:     config.Jobs{Webhooks: config.WebhookConfig{Command: "curl https://hook/?key=webhook-key"}},
		Metadata: config.Metadata{Providers: map[string]config.Provider{"acoustid": {Enabled: true, Secret: &secret}}},
		Downloaders: config.Downloaders{Plugins: []config.PluginConfig{
			{Name: "deezer", Config: map[string]any{"arl": "plugin-arl"}},
		}},
	}

	raw, err := json.Marshal(redactConfig(cfg))
	if err != nil {
		t.Fatal(err)
	}
	for _, leaked := range []string{"telegram-token", "webhook-key", "provider-secret", "plugin-arl"} {
		if strings.Contains(string(raw), leaked) {
			t.Errorf("redacted config still contains %q: %s", leaked, raw)
		}
	}

	// The live config must be untouched.
	if cfg.Telegram.Token != "telegram-token" || *cfg.Metadata.Providers["acoustid"].Secret != "provider-secret" || cfg.Downloaders.Plugins[0].Config["arl"] != "plugin-arl" {
		t.Error("redactConfig mutated the live config")
	}
}

func TestApplySettingRejectsUnsafeKeys(t *testing.T) {
	for _, key := range []string{"libraryPath", "downloadPath", "jobs.log_path", "telegram.token", "telegram.allowedUsers", "metadata.providers.acoustid.secret"} {
		cfg := &config.Config{Metadata: config.Metadata{Providers: map[string]config.Provider{}}}
		if err := applySetting(cfg, key, "x"); err == nil {
			t.Errorf("expected %q to be rejected", key)
		}
	}
}

func TestApplySettingPathTemplates(t *testing.T) {
	for _, tc := range []struct {
		key, value string
		ok         bool
	}{
		{"import.paths.default_path", "%asciify{$albumartist}/$album/$title", true},
		{"import.paths.default_path", "", false},
		{"import.paths.default_path", "../outside/$title", false},
		{"import.paths.album:ep", "$albumartist/../../$title", false},
		{"import.paths.album:single", "/etc/$title", false},
		{"import.paths.compilations", "", true},
	} {
		cfg := &config.Config{}
		err := applySetting(cfg, tc.key, tc.value)
		if (err == nil) != tc.ok {
			t.Errorf("applySetting(%q, %q): ok=%v, err=%v", tc.key, tc.value, tc.ok, err)
		}
	}
}

func TestApplySettingRejectsUnknownProviders(t *testing.T) {
	cfg := &config.Config{
		Metadata: config.Metadata{Providers: map[string]config.Provider{"deezer": {}}},
		Lyrics:   config.Lyrics{Providers: map[string]config.LyricsProvider{"lrclib": {}}},
	}
	for _, key := range []string{"metadata.providers.typo.enabled", "lyrics.providers.typo.enabled"} {
		if err := applySetting(cfg, key, "true"); err == nil {
			t.Errorf("expected %q to be rejected", key)
		}
	}
	if len(cfg.Metadata.Providers) != 1 || len(cfg.Lyrics.Providers) != 1 {
		t.Error("unknown providers must not be added to the config")
	}
	if err := applySetting(cfg, "metadata.providers.deezer.enabled", "true"); err != nil || !cfg.Metadata.Providers["deezer"].Enabled {
		t.Errorf("expected known provider to be updated, err=%v", err)
	}
}
