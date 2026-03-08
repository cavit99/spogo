package cli

type AuthCmd struct {
	Status AuthStatusCmd `kong:"cmd,help='Show cookie status.'"`
	Import AuthImportCmd `kong:"cmd,help='Import browser cookies.'"`
	Paste  AuthPasteCmd  `kong:"cmd,help='Paste cookie values from the browser.'"`
	Clear  AuthClearCmd  `kong:"cmd,help='Clear stored cookies.'"`
}

type AuthStatusCmd struct{}

type AuthImportCmd struct {
	Browser    string `help:"Browser name (chrome|brave|edge|firefox|safari)."`
	Profile    string `name:"browser-profile" help:"Browser profile name."`
	CookiePath string `help:"Cookie cache file path."`
	Domain     string `help:"Cookie domain suffix." default:"spotify.com"`
}

type AuthPasteCmd struct {
	CookiePath string `help:"Cookie cache file path."`
	Domain     string `help:"Cookie domain suffix." default:"spotify.com"`
	Path       string `help:"Cookie path." default:"/"`
}

type AuthClearCmd struct{}

type authStatusPayload struct {
	CookieCount int    `json:"cookie_count"`
	HasSPDC     bool   `json:"has_sp_dc"`
	HasSPT      bool   `json:"has_sp_t"`
	HasSPKey    bool   `json:"has_sp_key"`
	Source      string `json:"source"`
}
