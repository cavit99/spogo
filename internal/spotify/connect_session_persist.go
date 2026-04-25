package spotify

// Bootstrap-state cache, persisted across spogo invocations.
//
// Without this cache, every CLI invocation re-runs the full bootstrap on
// connectSession (access token fetch, appServerConfig scrape, client-token
// fetch) — about 1s of unavoidable HTTP. Caching what's safely
// derivable-and-validatable cuts that to a single sp_dc cookie read on a
// warm cold-start.
//
// What we cache:
//   - access token + expiry (already has expiry; validate before use)
//   - client token + expiry (~30 min ttl)
//   - clientID (stable for the account, but bound to access token; fetched
//     together)
//   - clientVersion (e.g. "harmony:4.43.2-..."; rotates when Spotify ships
//     web-player updates — we add a 24h ceiling so a stale value can't
//     linger forever)
//   - connectDeviceID (random hex; harmless if reused, deduplicates the
//     "merid-bridge" Spotify-side identity across invocations)
//
// What we DO NOT cache:
//   - sp_t / sp_dc cookie values — we re-read those from cookies anyway
//   - dealer connection ID + bridge registration — bound to a WebSocket
//     that closes on process exit; reusing a stale one means a 4xx on the
//     next call. Cheaper to re-establish per process (~1s).
//
// Invalidation:
//   - sp cookie fingerprint mismatch → cookies were rotated, account
//     switched, or imported anew. Drop everything and bootstrap fresh.
//   - any persisted-token expiry past now → drop just that field, fetch
//     fresh.
//   - clientVersion older than 24h → re-scrape appServerConfig.
//   - 401/403 on a downstream call → call sites already invalidate via
//     the existing token-refresh path; we'll observe the next bootstrap.
//
// Persistence is best-effort: load failures are silent (cold-bootstrap
// like before), save failures are silent (next run will redo bootstrap,
// no functional impact). The file is written atomically (`tmp` + rename)
// to avoid races between concurrent spogo invocations.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	persistedSessionVersion        = 1
	persistedClientVersionMaxAgeHr = 24
)

type persistedSession struct {
	Version              int       `json:"version"`
	SavedAt              time.Time `json:"savedAt"`
	CookieFingerprint    string    `json:"cookieFingerprint,omitempty"`
	AccessToken          string    `json:"accessToken,omitempty"`
	AccessTokenExpiresAt time.Time `json:"accessTokenExpiresAt,omitempty"`
	ClientToken          string    `json:"clientToken,omitempty"`
	ClientTokenExpiresAt time.Time `json:"clientTokenExpiresAt,omitempty"`
	ClientID             string    `json:"clientID,omitempty"`
	ClientVersion        string    `json:"clientVersion,omitempty"`
	ClientVersionAt      time.Time `json:"clientVersionAt,omitempty"`
	ConnectDeviceID      string    `json:"connectDeviceID,omitempty"`
}

// loadPersistedSession reads a session-state file and returns whatever
// can still be trusted. Anything stale or fingerprint-mismatched is
// dropped silently so the caller can re-bootstrap exactly the missing
// pieces. cookieFingerprint should be the live fingerprint computed from
// the current cookie source; mismatches drop everything.
func loadPersistedSession(path, cookieFingerprint string) persistedSession {
	if path == "" {
		return persistedSession{}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return persistedSession{}
	}
	var p persistedSession
	if err := json.Unmarshal(data, &p); err != nil {
		return persistedSession{}
	}
	if p.Version != persistedSessionVersion {
		return persistedSession{}
	}
	if cookieFingerprint != "" && p.CookieFingerprint != "" && p.CookieFingerprint != cookieFingerprint {
		return persistedSession{}
	}
	now := time.Now()
	if !p.AccessTokenExpiresAt.IsZero() && now.After(p.AccessTokenExpiresAt.Add(-time.Minute)) {
		p.AccessToken = ""
		p.AccessTokenExpiresAt = time.Time{}
	}
	if !p.ClientTokenExpiresAt.IsZero() && now.After(p.ClientTokenExpiresAt.Add(-time.Minute)) {
		p.ClientToken = ""
		p.ClientTokenExpiresAt = time.Time{}
	}
	if !p.ClientVersionAt.IsZero() && now.Sub(p.ClientVersionAt) > persistedClientVersionMaxAgeHr*time.Hour {
		p.ClientVersion = ""
		p.ClientVersionAt = time.Time{}
	}
	return p
}

func savePersistedSession(path string, p persistedSession) {
	if path == "" {
		return
	}
	p.Version = persistedSessionVersion
	p.SavedAt = time.Now()
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return
	}
	tmp, err := os.CreateTemp(dir, ".state-*.json.tmp")
	if err != nil {
		return
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return
	}
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return
	}
	_ = os.Rename(tmpPath, path)
}

// cookieFingerprint hashes the sp_dc + sp_t pair so we can detect when a
// cookie reimport / account switch has invalidated everything else. Only
// these two cookies meaningfully bind the session; ignoring the rest
// keeps the fingerprint stable across noise.
func cookieFingerprint(cookies []*http.Cookie) string {
	var spDC, spT string
	for _, cookie := range cookies {
		switch cookie.Name {
		case "sp_dc":
			spDC = cookie.Value
		case "sp_t":
			spT = cookie.Value
		}
	}
	if spDC == "" && spT == "" {
		return ""
	}
	h := sha256.Sum256([]byte("sp_dc:" + spDC + "|sp_t:" + spT))
	return hex.EncodeToString(h[:8])
}
