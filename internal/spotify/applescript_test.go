//go:build darwin
// +build darwin

package spotify

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestAppleScriptPlaybackControls(t *testing.T) {
	ctx := context.Background()
	var scripts []string
	client := &AppleScriptClient{
		run: func(_ context.Context, script string) (string, error) {
			scripts = append(scripts, script)
			return "", nil
		},
	}

	tests := []struct {
		name string
		run  func() error
		want string
	}{
		{
			name: "play current",
			run:  func() error { return client.Play(ctx, "") },
			want: `tell application "Spotify" to play`,
		},
		{
			name: "play uri",
			run:  func() error { return client.Play(ctx, "spotify:track:1") },
			want: `tell application "Spotify" to play track "spotify:track:1"`,
		},
		{
			name: "pause",
			run:  func() error { return client.Pause(ctx) },
			want: `tell application "Spotify" to pause`,
		},
		{
			name: "next",
			run:  func() error { return client.Next(ctx) },
			want: `tell application "Spotify" to next track`,
		},
		{
			name: "previous",
			run:  func() error { return client.Previous(ctx) },
			want: `tell application "Spotify" to previous track`,
		},
		{
			name: "seek",
			run:  func() error { return client.Seek(ctx, 42500) },
			want: `tell application "Spotify" to set player position to 42`,
		},
		{
			name: "volume",
			run:  func() error { return client.Volume(ctx, 67) },
			want: `tell application "Spotify" to set sound volume to 67`,
		},
		{
			name: "shuffle on",
			run:  func() error { return client.Shuffle(ctx, true) },
			want: `tell application "Spotify" to set shuffling to true`,
		},
		{
			name: "shuffle off",
			run:  func() error { return client.Shuffle(ctx, false) },
			want: `tell application "Spotify" to set shuffling to false`,
		},
		{
			name: "repeat context",
			run:  func() error { return client.Repeat(ctx, "context") },
			want: `tell application "Spotify" to set repeating to true`,
		},
		{
			name: "repeat track",
			run:  func() error { return client.Repeat(ctx, "track") },
			want: `tell application "Spotify" to set repeating to true`,
		},
		{
			name: "repeat off",
			run:  func() error { return client.Repeat(ctx, "off") },
			want: `tell application "Spotify" to set repeating to false`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scripts = nil
			if err := tt.run(); err != nil {
				t.Fatalf("run: %v", err)
			}
			if len(scripts) != 1 || scripts[0] != tt.want {
				t.Fatalf("script = %#v, want %q", scripts, tt.want)
			}
		})
	}
}

func TestAppleScriptRunScriptUsesInjectedRunner(t *testing.T) {
	errBoom := errors.New("boom")
	client := &AppleScriptClient{
		run: func(context.Context, string) (string, error) {
			return "", errBoom
		},
	}

	if err := client.Pause(context.Background()); !errors.Is(err, errBoom) {
		t.Fatalf("Pause error = %v, want %v", err, errBoom)
	}
}

func TestAppleScriptPlayback(t *testing.T) {
	client := &AppleScriptClient{
		run: func(_ context.Context, script string) (string, error) {
			if !strings.Contains(script, "current track") {
				t.Fatalf("unexpected script: %s", script)
			}
			return "Song|||Artist|||Album|||spotify:track:1|||180000|||42.5|||playing|||55|||true|||true", nil
		},
	}

	status, err := client.Playback(context.Background())
	if err != nil {
		t.Fatalf("Playback: %v", err)
	}
	if !status.IsPlaying || status.ProgressMS != 42500 || !status.Shuffle || status.Repeat != "context" {
		t.Fatalf("unexpected status: %#v", status)
	}
	if status.Item == nil || status.Item.URI != "spotify:track:1" || status.Item.DurationMS != 180000 {
		t.Fatalf("unexpected item: %#v", status.Item)
	}
	if status.Device.ID != "local" || status.Device.Volume != 55 || !status.Device.Active {
		t.Fatalf("unexpected device: %#v", status.Device)
	}
}

func TestAppleScriptPlaybackBadOutput(t *testing.T) {
	client := &AppleScriptClient{
		run: func(context.Context, string) (string, error) {
			return "bad output", nil
		},
	}

	_, err := client.Playback(context.Background())
	if err == nil || !strings.Contains(err.Error(), "unexpected applescript output") {
		t.Fatalf("Playback error = %v, want unexpected output", err)
	}
}

func TestAppleScriptLocalDeviceAndTransfer(t *testing.T) {
	client := &AppleScriptClient{}

	devices, err := client.Devices(context.Background())
	if err != nil {
		t.Fatalf("Devices: %v", err)
	}
	if len(devices) != 1 || devices[0].ID != "local" || !devices[0].Active {
		t.Fatalf("unexpected devices: %#v", devices)
	}
	if err := client.Transfer(context.Background(), "device"); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("Transfer error = %v, want ErrUnsupported", err)
	}
}

func TestAppleScriptUnsupportedWithoutFallback(t *testing.T) {
	ctx := context.Background()
	client := &AppleScriptClient{}

	if err := client.QueueAdd(ctx, "spotify:track:1"); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("QueueAdd error = %v", err)
	}
	if _, err := client.Queue(ctx); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("Queue error = %v", err)
	}
	if _, err := client.Search(ctx, "track", "q", 1, 0); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("Search error = %v", err)
	}
	if _, err := client.GetTrack(ctx, "track"); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("GetTrack error = %v", err)
	}
	if _, err := client.GetAlbum(ctx, "album"); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("GetAlbum error = %v", err)
	}
	if _, err := client.GetArtist(ctx, "artist"); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("GetArtist error = %v", err)
	}
	if _, err := client.GetPlaylist(ctx, "playlist"); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("GetPlaylist error = %v", err)
	}
	if _, err := client.GetShow(ctx, "show"); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("GetShow error = %v", err)
	}
	if _, err := client.GetEpisode(ctx, "episode"); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("GetEpisode error = %v", err)
	}
	if _, _, err := client.LibraryTracks(ctx, 1, 0); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("LibraryTracks error = %v", err)
	}
	if _, _, err := client.LibraryAlbums(ctx, 1, 0); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("LibraryAlbums error = %v", err)
	}
	if err := client.LibraryModify(ctx, "tracks", []string{"id"}, "PUT"); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("LibraryModify error = %v", err)
	}
	if err := client.FollowArtists(ctx, []string{"id"}, "PUT"); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("FollowArtists error = %v", err)
	}
	if _, _, _, err := client.FollowedArtists(ctx, 1, ""); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("FollowedArtists error = %v", err)
	}
	if _, _, err := client.Playlists(ctx, 1, 0); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("Playlists error = %v", err)
	}
	if _, _, err := client.PlaylistTracks(ctx, "playlist", 1, 0); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("PlaylistTracks error = %v", err)
	}
	if _, err := client.CreatePlaylist(ctx, "mix", true, false); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("CreatePlaylist error = %v", err)
	}
	if err := client.AddTracks(ctx, "playlist", []string{"uri"}); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("AddTracks error = %v", err)
	}
	if err := client.RemoveTracks(ctx, "playlist", []string{"uri"}); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("RemoveTracks error = %v", err)
	}
}

func TestAppleScriptDelegatesToFallback(t *testing.T) {
	ctx := context.Background()
	calls := map[string]int{}
	client := &AppleScriptClient{fallback: apiStub{calls: calls}}

	_ = client.QueueAdd(ctx, "spotify:track:1")
	_, _ = client.Queue(ctx)
	_, _ = client.Search(ctx, "track", "q", 1, 0)
	_, _ = client.GetTrack(ctx, "track")
	_, _ = client.GetAlbum(ctx, "album")
	_, _ = client.GetArtist(ctx, "artist")
	_, _ = client.GetPlaylist(ctx, "playlist")
	_, _ = client.GetShow(ctx, "show")
	_, _ = client.GetEpisode(ctx, "episode")
	_, _, _ = client.LibraryTracks(ctx, 1, 0)
	_, _, _ = client.LibraryAlbums(ctx, 1, 0)
	_ = client.LibraryModify(ctx, "tracks", []string{"id"}, "PUT")
	_ = client.FollowArtists(ctx, []string{"id"}, "PUT")
	_, _, _, _ = client.FollowedArtists(ctx, 1, "")
	_, _, _ = client.Playlists(ctx, 1, 0)
	_, _, _ = client.PlaylistTracks(ctx, "playlist", 1, 0)
	_, _ = client.CreatePlaylist(ctx, "mix", true, false)
	_ = client.AddTracks(ctx, "playlist", []string{"uri"})
	_ = client.RemoveTracks(ctx, "playlist", []string{"uri"})

	for _, name := range []string{
		"QueueAdd",
		"Queue",
		"Search",
		"GetTrack",
		"GetAlbum",
		"GetArtist",
		"GetPlaylist",
		"GetShow",
		"GetEpisode",
		"LibraryTracks",
		"LibraryAlbums",
		"LibraryModify",
		"FollowArtists",
		"FollowedArtists",
		"Playlists",
		"PlaylistTracks",
		"CreatePlaylist",
		"AddTracks",
		"RemoveTracks",
	} {
		if calls[name] != 1 {
			t.Fatalf("%s calls = %d, want 1", name, calls[name])
		}
	}
}
