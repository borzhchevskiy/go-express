package GoExpress

import (
	"time"
	"strings"
)

type Cookie struct {
	Name       string
	Value      string
	Path       string
	Domain     string
	Expires    time.Time
	MaxAge     string
	Secure     bool
	HttpOnly   bool
}

// Cookie.String() returns a string, put it to Set-Cookie header
func (c *Cookie) String() string {
	var b strings.Builder
	b.WriteString(c.Name)
	b.WriteRune('=')
	b.WriteString(c.Value)
	if len(c.Path) > 0 {
		b.WriteString("; Path=")
		b.WriteString(c.Path)
	}
	if len(c.Domain) > 0 {
		b.WriteString("; Domain=")
		b.WriteString(c.Domain)
	}
	b.WriteString("; Expires=")
	b.WriteString(c.Expires.In(time.FixedZone("GMT", 0)).Format(time.RFC1123))
	if c.MaxAge != "0" {
		b.WriteString("; Max-Age=")
		b.WriteString(c.MaxAge)
	}
	if c.HttpOnly {
		b.WriteString("; HttpOnly")
	}
	if c.Secure {
		b.WriteString("; Secure")
	}
	return b.String()
}