package cli

import (
	"fmt"
	"net/http"

	"github.com/steipete/spogo/internal/app"
)

func (cmd *AuthStatusCmd) Run(ctx *app.Context) error {
	cookiesList, sourceLabel, err := readCookies(ctx)
	if err != nil {
		return err
	}
	payload := authStatusPayload{
		CookieCount: len(cookiesList),
		HasSPDC:     hasCookie(cookiesList, "sp_dc"),
		HasSPT:      hasCookie(cookiesList, "sp_t"),
		HasSPKey:    hasCookie(cookiesList, "sp_key"),
		Source:      sourceLabel,
	}
	plain := []string{fmt.Sprintf("%d\t%t\t%t\t%t\t%s", payload.CookieCount, payload.HasSPDC, payload.HasSPT, payload.HasSPKey, payload.Source)}
	human := []string{fmt.Sprintf("Cookies: %d (%s)", payload.CookieCount, payload.Source)}
	human = append(human, cookieStatusLine("Session cookie", "sp_dc", payload.HasSPDC, ""))
	human = append(human, cookieStatusLine("Device cookie", "sp_t", payload.HasSPT, "needed for connect playback"))
	if payload.HasSPKey {
		human = append(human, "Optional cookie: sp_key")
	}
	return ctx.Output.Emit(payload, plain, human)
}

func hasCookie(cookiesList []*http.Cookie, name string) bool {
	for _, cookie := range cookiesList {
		if cookie.Name == name {
			return true
		}
	}
	return false
}

func cookieStatusLine(label, name string, present bool, missingHint string) string {
	if present {
		return fmt.Sprintf("%s: %s", label, name)
	}
	line := fmt.Sprintf("%s: missing %s", label, name)
	if missingHint != "" {
		line += " (" + missingHint + ")"
	}
	return line
}
