package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/steipete/spogo/internal/config"
	"github.com/steipete/spogo/internal/cookies"
	"github.com/steipete/spogo/internal/output"
	"github.com/steipete/spogo/internal/spotify"
)

func TestNewContextLoadsProfile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	cfg := config.Default()
	cfg.SetProfile("work", config.Profile{Market: "US", Language: "en"})
	cfg.DefaultProfile = "work"
	if err := config.Save(path, cfg); err != nil {
		t.Fatalf("save: %v", err)
	}
	ctx, err := NewContext(Settings{ConfigPath: path, Format: output.FormatPlain})
	if err != nil {
		t.Fatalf("new context: %v", err)
	}
	if ctx.Profile.Market != "US" {
		t.Fatalf("market: %s", ctx.Profile.Market)
	}
	if ctx.ProfileKey != "work" {
		t.Fatalf("profile key: %s", ctx.ProfileKey)
	}
}

func TestResolveCookiePath(t *testing.T) {
	ctx := &Context{ConfigPath: "/tmp/spogo/config.toml", ProfileKey: "default"}
	path := ctx.ResolveCookiePath()
	if filepath.Base(path) != "default.json" {
		t.Fatalf("cookie path: %s", path)
	}
}

func TestValidateProfile(t *testing.T) {
	ctx := &Context{Profile: config.Profile{Market: "USA"}}
	if err := ctx.ValidateProfile(); err == nil {
		t.Fatalf("expected error")
	}
}

func TestEnsureTimeout(t *testing.T) {
	ctx := &Context{Settings: Settings{}}
	if ctx.EnsureTimeout() == 0 {
		t.Fatalf("expected default timeout")
	}
	ctx = &Context{Settings: Settings{Timeout: time.Second}}
	if ctx.EnsureTimeout() != time.Second {
		t.Fatalf("expected custom timeout")
	}
}

func TestCommandContext(t *testing.T) {
	type contextKey string

	var nilCtx *Context
	if nilCtx.CommandContext() == nil {
		t.Fatalf("expected background context for nil receiver")
	}

	ctx := &Context{}
	if ctx.CommandContext() == nil {
		t.Fatalf("expected background context")
	}

	const key contextKey = "key"
	custom := context.WithValue(context.Background(), key, "value")
	ctx.SetCommandContext(custom)
	if got := ctx.CommandContext().Value(key); got != "value" {
		t.Fatalf("context value = %v", got)
	}

	var nilCommandCtx context.Context
	ctx.SetCommandContext(nilCommandCtx)
	if ctx.CommandContext() == nil {
		t.Fatalf("expected background context after nil set")
	}

	nilCtx.SetCommandContext(context.Background())
}

func TestSpotifyCachedClient(t *testing.T) {
	ctx := &Context{}
	ctx.SetSpotify(dummySpotify{})
	client, err := ctx.Spotify()
	if err != nil {
		t.Fatalf("spotify: %v", err)
	}
	if _, ok := client.(dummySpotify); !ok {
		t.Fatalf("expected cached client")
	}
}

func TestSaveProfile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	cfg := config.Default()
	ctx := &Context{Config: cfg, ConfigPath: path, ProfileKey: "default"}
	if err := ctx.SaveProfile(config.Profile{Market: "US"}); err != nil {
		t.Fatalf("save: %v", err)
	}
	loaded, err := config.Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.Profile("default").Market != "US" {
		t.Fatalf("profile not saved")
	}
}

func TestSaveProfileNilContext(t *testing.T) {
	var ctx *Context
	if err := ctx.SaveProfile(config.Profile{Market: "US"}); err == nil {
		t.Fatalf("expected error")
	}
}

func TestSaveProfileNilConfig(t *testing.T) {
	ctx := &Context{}
	if err := ctx.SaveProfile(config.Profile{Market: "US"}); err == nil {
		t.Fatalf("expected error")
	}
}

func TestCookieSourceFile(t *testing.T) {
	ctx := &Context{Profile: config.Profile{CookiePath: "/tmp/cookies.json"}}
	src, err := ctx.cookieSource()
	if err != nil {
		t.Fatalf("cookie source: %v", err)
	}
	if _, ok := src.(cookies.FileSource); !ok {
		t.Fatalf("expected file source")
	}
}

func TestCookieSourceBrowser(t *testing.T) {
	ctx := &Context{Profile: config.Profile{Browser: "chrome"}}
	src, err := ctx.cookieSource()
	if err != nil {
		t.Fatalf("cookie source: %v", err)
	}
	if _, ok := src.(cookies.BrowserSource); !ok {
		t.Fatalf("expected browser source")
	}
}

func TestCookieSourceDefaultBrowser(t *testing.T) {
	ctx := &Context{Profile: config.Profile{}}
	src, err := ctx.cookieSource()
	if err != nil {
		t.Fatalf("cookie source: %v", err)
	}
	browser, ok := src.(cookies.BrowserSource)
	if !ok || browser.Browser != "chrome" {
		t.Fatalf("expected chrome source")
	}
}

func TestCookieSourceDefaultFile(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")
	ctx := &Context{ConfigPath: configPath, ProfileKey: "default"}
	cookiePath := config.CookiePath(configPath, "default")
	if err := os.MkdirAll(filepath.Dir(cookiePath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(cookiePath, []byte("[]"), 0o644); err != nil {
		t.Fatalf("write cookie file: %v", err)
	}

	src, err := ctx.cookieSource()
	if err != nil {
		t.Fatalf("cookie source: %v", err)
	}
	fileSource, ok := src.(cookies.FileSource)
	if !ok || fileSource.Path != cookiePath {
		t.Fatalf("expected default file source, got %#v", src)
	}
}

func TestSpotifyNilContext(t *testing.T) {
	var ctx *Context
	if _, err := ctx.Spotify(); err == nil {
		t.Fatalf("expected error")
	}
}

func TestSpotifyBuildsClient(t *testing.T) {
	ctx := &Context{Profile: config.Profile{CookiePath: "/tmp/cookies.json"}}
	client, err := ctx.Spotify()
	if err != nil {
		t.Fatalf("spotify: %v", err)
	}
	if client == nil {
		t.Fatalf("expected client")
	}
}

func TestSpotifyWebEngine(t *testing.T) {
	ctx := &Context{Profile: config.Profile{CookiePath: "/tmp/cookies.json", Engine: "web"}}
	client, err := ctx.Spotify()
	if err != nil {
		t.Fatalf("spotify: %v", err)
	}
	if client == nil {
		t.Fatalf("expected client")
	}
}

func TestSpotifyAutoEngine(t *testing.T) {
	ctx := &Context{Profile: config.Profile{CookiePath: "/tmp/cookies.json", Engine: "auto"}}
	client, err := ctx.Spotify()
	if err != nil {
		t.Fatalf("spotify: %v", err)
	}
	if client == nil {
		t.Fatalf("expected client")
	}
}

func TestSpotifyUnknownEngine(t *testing.T) {
	ctx := &Context{Profile: config.Profile{CookiePath: "/tmp/cookies.json", Engine: "nope"}}
	if _, err := ctx.Spotify(); err == nil {
		t.Fatalf("expected error")
	}
}

func TestIsColorEnabled(t *testing.T) {
	if isColorEnabled(output.FormatJSON, false) {
		t.Fatalf("expected false")
	}
	if isColorEnabled(output.FormatHuman, true) {
		t.Fatalf("expected false")
	}
	t.Setenv("NO_COLOR", "1")
	if isColorEnabled(output.FormatHuman, false) {
		t.Fatalf("expected false")
	}
	t.Setenv("NO_COLOR", "")
	t.Setenv("TERM", "dumb")
	if isColorEnabled(output.FormatHuman, false) {
		t.Fatalf("expected false")
	}
}

func TestNewContextInvalidFormat(t *testing.T) {
	_, err := NewContext(Settings{Format: "bad"})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestNewContextInvalidConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.toml")
	if err := os.WriteFile(path, []byte("not=toml=\""), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := NewContext(Settings{ConfigPath: path, Format: output.FormatPlain}); err == nil {
		t.Fatalf("expected error")
	}
}

func TestSetSpotifyNilContext(t *testing.T) {
	var ctx *Context
	ctx.SetSpotify(dummySpotify{})
}

func TestSetSpotify(t *testing.T) {
	ctx := &Context{}
	ctx.SetSpotify(dummySpotify{})
	if ctx.spotifyClient == nil {
		t.Fatalf("expected spotify client")
	}
}

func TestValidateProfileOK(t *testing.T) {
	ctx := &Context{Profile: config.Profile{Market: "US"}}
	if err := ctx.ValidateProfile(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveProfileKey(t *testing.T) {
	cfg := &config.Config{DefaultProfile: "work"}
	if got := resolveProfileKey(cfg, "personal"); got != "personal" {
		t.Fatalf("requested profile = %q", got)
	}
	if got := resolveProfileKey(cfg, ""); got != "work" {
		t.Fatalf("default profile = %q", got)
	}
	if got := resolveProfileKey(nil, ""); got != config.DefaultProfile {
		t.Fatalf("nil config profile = %q", got)
	}
}

func TestApplySettingsOverrides(t *testing.T) {
	profile := applySettings(config.Profile{
		Market:   "GB",
		Language: "en",
		Device:   "desk",
		Engine:   "connect",
	}, Settings{
		Market:   "US",
		Language: "tr",
		Device:   "phone",
		Engine:   "web",
	})
	if profile.Market != "US" || profile.Language != "tr" || profile.Device != "phone" || profile.Engine != "web" {
		t.Fatalf("settings not applied: %#v", profile)
	}
}

func TestNewOutputWriterDefaultFormat(t *testing.T) {
	writer, err := newOutputWriter(Settings{})
	if err != nil {
		t.Fatalf("new output writer: %v", err)
	}
	if writer == nil {
		t.Fatalf("expected writer")
	}
}

type dummySpotify struct{}

func (dummySpotify) Search(context.Context, string, string, int, int) (spotify.SearchResult, error) {
	return spotify.SearchResult{}, nil
}

func (dummySpotify) GetTrack(context.Context, string) (spotify.Item, error) {
	return spotify.Item{}, nil
}

func (dummySpotify) GetAlbum(context.Context, string) (spotify.Item, error) {
	return spotify.Item{}, nil
}

func (dummySpotify) GetArtist(context.Context, string) (spotify.Item, error) {
	return spotify.Item{}, nil
}

func (dummySpotify) GetPlaylist(context.Context, string) (spotify.Item, error) {
	return spotify.Item{}, nil
}

func (dummySpotify) GetShow(context.Context, string) (spotify.Item, error) {
	return spotify.Item{}, nil
}

func (dummySpotify) GetEpisode(context.Context, string) (spotify.Item, error) {
	return spotify.Item{}, nil
}

func (dummySpotify) Playback(context.Context) (spotify.PlaybackStatus, error) {
	return spotify.PlaybackStatus{}, nil
}
func (dummySpotify) Play(context.Context, string) error                { return nil }
func (dummySpotify) Pause(context.Context) error                       { return nil }
func (dummySpotify) Next(context.Context) error                        { return nil }
func (dummySpotify) Previous(context.Context) error                    { return nil }
func (dummySpotify) Seek(context.Context, int) error                   { return nil }
func (dummySpotify) Volume(context.Context, int) error                 { return nil }
func (dummySpotify) Shuffle(context.Context, bool) error               { return nil }
func (dummySpotify) Repeat(context.Context, string) error              { return nil }
func (dummySpotify) Devices(context.Context) ([]spotify.Device, error) { return nil, nil }
func (dummySpotify) Transfer(context.Context, string) error            { return nil }
func (dummySpotify) QueueAdd(context.Context, string) error            { return nil }
func (dummySpotify) Queue(context.Context) (spotify.Queue, error)      { return spotify.Queue{}, nil }
func (dummySpotify) LibraryTracks(context.Context, int, int) ([]spotify.Item, int, error) {
	return nil, 0, nil
}

func (dummySpotify) LibraryAlbums(context.Context, int, int) ([]spotify.Item, int, error) {
	return nil, 0, nil
}
func (dummySpotify) LibraryModify(context.Context, string, []string, string) error { return nil }
func (dummySpotify) FollowArtists(context.Context, []string, string) error         { return nil }
func (dummySpotify) FollowedArtists(context.Context, int, string) ([]spotify.Item, int, string, error) {
	return nil, 0, "", nil
}

func (dummySpotify) Playlists(context.Context, int, int) ([]spotify.Item, int, error) {
	return nil, 0, nil
}

func (dummySpotify) PlaylistTracks(context.Context, string, int, int) ([]spotify.Item, int, error) {
	return nil, 0, nil
}

func (dummySpotify) CreatePlaylist(context.Context, string, bool, bool) (spotify.Item, error) {
	return spotify.Item{}, nil
}
func (dummySpotify) AddTracks(context.Context, string, []string) error    { return nil }
func (dummySpotify) RemoveTracks(context.Context, string, []string) error { return nil }
