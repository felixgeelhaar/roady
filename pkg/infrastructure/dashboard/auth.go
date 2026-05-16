package dashboard

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

// EnableAuthToken protects every dashboard request with a shared token. Pass
// an empty string to leave the server public. The token is accepted on three
// surfaces:
//   - Authorization: Bearer <token>
//   - Cookie:        roady_token=<token>
//   - Query param:   ?token=<token>  (one-time, sets the cookie + redirects)
//
// Token comparison is constant-time.
func (s *Server) EnableAuthToken(token string) {
	s.authToken = token
}

const authCookieName = "roady_token"

// authMiddleware enforces the shared token. The handler short-circuits with
// 401 unless one of the accepted credentials matches. A successful
// ?token=<v> handshake sets the cookie + redirects to strip the secret from
// the URL bar.
func authMiddleware(token string, next http.Handler) http.Handler {
	tokenBytes := []byte(token)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. Query-param handshake (one-time): set cookie + redirect without param.
		if q := r.URL.Query().Get("token"); q != "" {
			if constantTimeEqual(q, token, tokenBytes) {
				http.SetCookie(w, &http.Cookie{
					Name:     authCookieName,
					Value:    q,
					Path:     "/",
					HttpOnly: true,
					Secure:   r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https"),
					SameSite: http.SameSiteLaxMode,
					MaxAge:   24 * 60 * 60 * 30, // 30 days
				})
				redirect := *r.URL
				q := redirect.Query()
				q.Del("token")
				redirect.RawQuery = q.Encode()
				http.Redirect(w, r, redirect.String(), http.StatusSeeOther)
				return
			}
			// fall through; bad token in query is treated as unauthenticated
		}

		// 2. Cookie.
		if c, err := r.Cookie(authCookieName); err == nil {
			if constantTimeEqual(c.Value, token, tokenBytes) {
				next.ServeHTTP(w, r)
				return
			}
		}

		// 3. Bearer.
		if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
			if constantTimeEqual(strings.TrimPrefix(h, "Bearer "), token, tokenBytes) {
				next.ServeHTTP(w, r)
				return
			}
		}

		w.Header().Set("WWW-Authenticate", `Bearer realm="roady-dashboard"`)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	})
}

func constantTimeEqual(got, _ string, want []byte) bool {
	gotB := []byte(got)
	if len(gotB) != len(want) {
		return false
	}
	return subtle.ConstantTimeCompare(gotB, want) == 1
}
