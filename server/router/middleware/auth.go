package middleware

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/go-chi/chi"
	sessionstore "github.com/porter-dev/porter/internal/auth"
)

// Auth implements the authorization functions
type Auth struct {
	store      *sessionstore.PGStore
	cookieName string
}

// NewAuth returns a new Auth instance
func NewAuth(
	store *sessionstore.PGStore,
	cookieName string,
) *Auth {
	return &Auth{store, cookieName}
}

// BasicAuthenticate just checks that a user is logged in
func (auth *Auth) BasicAuthenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if auth.isLoggedIn(r) {
			next.ServeHTTP(w, r)
		} else {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}

		return
	})
}

// IDLocation represents the location of the ID to use for authentication
type IDLocation uint

const (
	// URLParam location looks for {id} in the URL
	URLParam IDLocation = iota
	// BodyParam location looks for user_id in the body
	BodyParam
)

type bodyID struct {
	UserID uint64 `json:"user_id"`
}

// DoesUserIDMatch checks the id URL parameter and verifies that it matches
// the one stored in the session
func (auth *Auth) DoesUserIDMatch(next http.Handler, loc IDLocation) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var id uint64
		var err error

		if loc == URLParam {
			id, err = strconv.ParseUint(chi.URLParam(r, "id"), 0, 64)
		} else if loc == BodyParam {
			form := &bodyID{}
			body, _ := ioutil.ReadAll(r.Body)
			err = json.Unmarshal(body, form)
			id = form.UserID

			// need to create a new stream for the body
			r.Body = ioutil.NopCloser(bytes.NewReader(body))
		}

		if err == nil && auth.doesSessionMatchID(r, uint(id)) {
			next.ServeHTTP(w, r)
		} else {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}

		return
	})
}

// Helpers
func (auth *Auth) doesSessionMatchID(r *http.Request, id uint) bool {
	session, _ := auth.store.Get(r, auth.cookieName)

	if sessID, ok := session.Values["user_id"].(uint); !ok || sessID != id {
		return false
	}

	return true
}

func (auth *Auth) isLoggedIn(r *http.Request) bool {
	session, _ := auth.store.Get(r, auth.cookieName)

	if auth, ok := session.Values["authenticated"].(bool); !auth || !ok {
		return false
	}
	return true
}