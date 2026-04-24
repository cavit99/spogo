//go:build darwin
// +build darwin

package app

import (
	"testing"

	"github.com/steipete/spogo/internal/config"
)

func TestSpotifyAppleScriptEngineDarwin(t *testing.T) {
	ctx := &Context{Profile: config.Profile{CookiePath: "/tmp/cookies.json", Engine: "applescript"}}
	client, err := ctx.Spotify()
	if err != nil {
		t.Fatalf("spotify: %v", err)
	}
	if client == nil {
		t.Fatalf("expected client")
	}
}
