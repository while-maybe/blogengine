package middleware

import (
	"fmt"
	"net/http"
	"strings"
)

type CSP struct {
	isProd          bool
	cspHeaderString string
}

func NewCSP(isProd bool) *CSP {

	AllowedStyleSources := []string{"https://fonts.googleapis.com"}
	AllowedImageSources := []string{"https://images.unsplash.com"}
	AllowedFontSources := []string{"https://fonts.gstatic.com"}

	styleSources := strings.Join(AllowedStyleSources, " ")
	imageSources := strings.Join(AllowedImageSources, " ")
	fontSources := strings.Join(AllowedFontSources, " ")

	cspHeader := "default-src 'self'; " +
		"script-src 'self'; " +
		fmt.Sprintf("style-src 'self' 'unsafe-inline' %s; ", styleSources) +
		fmt.Sprintf("img-src 'self' data: %s; ", imageSources) +
		fmt.Sprintf("font-src 'self' %s; ", fontSources) +
		"connect-src 'self'; " +
		"frame-ancestors 'none'; " +
		"base-uri 'self'; " +
		"form-action 'self'"

	return &CSP{
		isProd:          isProd,
		cspHeaderString: cspHeader,
	}
}

func (c *CSP) Middleware() Middleware {

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Content Security Policy
			w.Header().Set("Content-Security-Policy", c.cspHeaderString)

			// HSTS
			if c.isProd {
				w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains; preload")
			}

			// Other security headers
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("X-XSS-Protection", "1; mode=block")
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

			next.ServeHTTP(w, r)
		})
	}
}
