package handler

import (
	"errors"
	"log"
	"sort"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/strelov1/freehire/internal/auth/oauth"
)

// ListOAuthProviders returns the names of enabled OAuth providers, so the SPA
// renders only usable sign-in buttons.
func (h *Handler) ListOAuthProviders(c *fiber.Ctx) error {
	names := make([]string, 0, len(h.oauth))
	for name := range h.oauth {
		names = append(names, name)
	}
	sort.Strings(names)
	return c.JSON(fiber.Map{"data": names})
}

// OAuthStart begins the authorization-code flow: it stores a fresh CSRF state
// in a short-lived cookie and redirects the browser to the provider's consent
// page carrying the same state.
func (h *Handler) OAuthStart(c *fiber.Ctx) error {
	p, ok := h.oauth[c.Params("provider")]
	if !ok {
		return fiber.NewError(fiber.StatusNotFound, "unknown provider")
	}

	state, err := oauth.NewState()
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to start sign-in")
	}
	oauth.SetStateCookie(c, state, h.cookieSecure)
	// Remember where the SPA wants the user back, so sign-in from a deep page
	// (e.g. a job detail) returns there. Sanitized to a same-origin path.
	oauth.SetReturnCookie(c, oauth.SafeReturnPath(c.Query("returnTo")), h.cookieSecure)
	return c.Redirect(p.AuthCodeURL(state), fiber.StatusFound)
}

// OAuthCallback completes the flow: verify the CSRF state, exchange the code
// for the provider identity, resolve (or create) the account, start the
// session, and send the browser back to the SPA. The callback is a top-level
// navigation, so every failure redirects with auth_error instead of rendering
// JSON; details go to the server log.
func (h *Handler) OAuthCallback(c *fiber.Ctx) error {
	p, ok := h.oauth[c.Params("provider")]
	if !ok {
		return fiber.NewError(fiber.StatusNotFound, "unknown provider")
	}

	// The state and return target are single-use: clear both cookies no matter
	// how the rest goes. Re-sanitize the return path defensively.
	cookieState := c.Cookies(oauth.StateCookieName)
	returnTo := oauth.SafeReturnPath(c.Cookies(oauth.ReturnCookieName))
	oauth.ClearStateCookie(c, h.cookieSecure)
	oauth.ClearReturnCookie(c, h.cookieSecure)

	state, code := c.Query("state"), c.Query("code")
	if state == "" || state != cookieState {
		return h.oauthFail(c, p.Name(), returnTo, errors.New("state mismatch"))
	}
	if code == "" {
		return h.oauthFail(c, p.Name(), returnTo, errors.New("missing code"))
	}

	identity, err := p.FetchIdentity(c.Context(), code)
	if err != nil {
		return h.oauthFail(c, p.Name(), returnTo, err)
	}

	userID, err := h.accounts.ResolveOAuthAccount(c.Context(), p.Name(), identity.ProviderUserID, identity.Email, identity.EmailVerified)
	if err != nil {
		return h.oauthFail(c, p.Name(), returnTo, err)
	}

	if err := h.setSession(c, userID); err != nil {
		return h.oauthFail(c, p.Name(), returnTo, err)
	}
	return c.Redirect(h.frontendOrigin+returnTo, fiber.StatusFound)
}

// oauthFail logs the failure server-side and sends the browser back to where
// sign-in started with the generic auth_error marker (never a JSON error
// page), so the dialog can reopen in place. returnTo is a sanitized path.
func (h *Handler) oauthFail(c *fiber.Ctx, provider, returnTo string, err error) error {
	log.Printf("oauth %s: sign-in failed: %v", provider, err)
	sep := "?"
	if strings.Contains(returnTo, "?") {
		sep = "&"
	}
	return c.Redirect(h.frontendOrigin+returnTo+sep+"auth_error=oauth", fiber.StatusFound)
}
