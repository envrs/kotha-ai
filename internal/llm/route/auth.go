package route

import (
	"net/http"
	"os"
	"strings"

	"github.com/kothagpt/kotha/internal/llm/schema"
)

func schemaErr(msg string) error {
	return schema.NewError(schema.CodeAuth, errMsg(msg))
}

type errMsg string

func (e errMsg) Error() string { return string(e) }

// Auth credentials attached to outbound requests.

type Scheme string

const (
	SchemeNone   Scheme = "none"
	SchemeAPIKey Scheme = "api_key"
	SchemeBearer Scheme = "bearer"
)

type Credentials struct {
	Scheme Scheme `json:"scheme"`
	Key    string `json:"-"`
	Header string `json:"header,omitempty"`
}

func NoAuth() Credentials { return Credentials{Scheme: SchemeNone} }

func APIKey(key, header string) Credentials {
	if header == "" {
		header = "x-api-key"
	}
	return Credentials{Scheme: SchemeAPIKey, Key: key, Header: header}
}

func Bearer(token string) Credentials {
	return Credentials{Scheme: SchemeBearer, Key: token, Header: "Authorization"}
}

// Apply sets auth headers on an HTTP request. Missing key with a
// non-none scheme is an error so misconfig fails fast.
func (c Credentials) Apply(req *http.Request) error {
	switch c.Scheme {
	case SchemeNone, "":
		return nil
	case SchemeAPIKey:
		if c.Key == "" {
			return schemaErr("missing api key")
		}
		req.Header.Set(c.Header, c.Key)
		return nil
	case SchemeBearer:
		if c.Key == "" {
			return schemaErr("missing bearer token")
		}
		req.Header.Set("Authorization", "Bearer "+c.Key)
		return nil
	default:
		return schemaErr("unknown auth scheme " + string(c.Scheme))
	}
}

// FromEnv builds APIKey credentials from an env var. Empty value yields
// NoAuth so local/mock endpoints work without configuration.
func FromEnv(envVar, header string) Credentials {
	key := strings.TrimSpace(os.Getenv(envVar))
	if key == "" {
		return NoAuth()
	}
	return APIKey(key, header)
}
