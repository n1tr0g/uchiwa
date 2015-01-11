package uchiwa

import (
	"net/http"
	"strings"
	"crypto/sha1"
	"encoding/base64"
	"github.com/palourde/logger"
)

type Auth struct {
	Type string
	Users map[string]string
}

func (a *Auth) httpauth() func(http.Handler) http.Handler {
	switch a.Type {
		case "simple":
			for k, v :=range a.Users {
				return SimpleBasicAuth(k,v)
			}
		case "htpasswd":
			fn := func(u,p string) bool {
				requiredPass, exists := a.Users[u]
				if !exists {
					logger.Warningf("No entry for %s.", u)
					return false
				}
				if requiredPass[:5] == "{SHA}" {
					d := sha1.New()
					d.Write([]byte(p))
					if requiredPass[5:] == base64.StdEncoding.EncodeToString(d.Sum(nil)) {
						return true
					}
				} else {
					logger.Warningf("Invalid htpasswd entry for %s. Must be a SHA entry.", u)
				}
				return false
			}
			opts := AuthOptions{
				Realm: "Restricted",
				AuthFunc: fn,
			}
			return CustomBasicAuth(opts)
	}
	// no auth by default
	fn := func(h http.Handler) http.Handler {
		return h
	}
	return fn
}

type Authenticate interface {
	httpauth() func(http.Handler) http.Handler
}

func authType(c *Config) Authenticate {
	a := new(Auth)
	a.Type = c.Uchiwa.Auth

	switch strings.ToLower(c.Uchiwa.Auth) {
		case "simple":
			users := make(map[string]string,1)
			users[c.Uchiwa.User] = c.Uchiwa.Pass
			a.Users = users
		case "htpasswd":
			a.Users = c.Uchiwa.Users
	}
	return a
}
