package client

import (
	"net/url"
	"strings"
)

const defaultVersionPrefix = "v1"
const defaultEndpoint = "https://api.hostaria.cloud"

func normalizeRoutePrefix(raw string) string {
	s := strings.Trim(strings.TrimSpace(raw), "/")
	if s == "" {
		return ""
	}
	if i := strings.IndexByte(s, '/'); i >= 0 {
		s = s[:i]
	}
	return s
}

func isAPIRoutePrefix(seg string) bool {
	if len(seg) < 2 || seg[0] != 'v' {
		return false
	}
	for i := 1; i < len(seg); i++ {
		if seg[i] < '0' || seg[i] > '9' {
			return false
		}
	}
	return true
}

func stripLegacyAPIPathSegment(path string) string {
	p := strings.Trim(strings.TrimSpace(path), "/")
	if p == "api" {
		return ""
	}
	if strings.HasSuffix(p, "/api") {
		return strings.Trim(strings.TrimSuffix(p, "/api"), "/")
	}
	return p
}

func stripLegacyAPIResource(resource string) string {
	r := strings.Trim(strings.TrimSpace(resource), "/")
	r = strings.TrimPrefix(r, "api/")
	r = strings.TrimPrefix(r, "v1/")
	for strings.HasPrefix(r, "v") {
		seg, rest, found := strings.Cut(r, "/")
		if !isAPIRoutePrefix(seg) {
			break
		}
		if !found {
			return "/"
		}
		r = rest
	}
	if r == "api" || r == "" {
		return "/"
	}
	return "/" + r
}

// JoinAPI builds {endpoint}/v1/{resource} without ever producing /v1/api or /v1/api/v1.
// endpoint may be the host ("https://api.hostaria.cloud") or already versioned ("…/v1").
func JoinAPI(endpoint, resource string) string {
	base := strings.TrimRight(strings.TrimSpace(endpoint), "/")
	resource = stripLegacyAPIResource(resource)
	if base == "" {
		return "/" + defaultVersionPrefix + resource
	}
	u, err := url.Parse(base)
	if err == nil && (u.Scheme == "http" || u.Scheme == "https") {
		p := stripLegacyAPIPathSegment(u.Path)
		u.Path = "/" + p
		u.RawPath = ""
		base = strings.TrimRight(u.String(), "/")
		if isAPIRoutePrefix(normalizeRoutePrefix(p)) {
			return base + resource
		}
		return base + "/" + defaultVersionPrefix + resource
	}
	if isAPIRoutePrefix(normalizeRoutePrefix(base)) {
		return base + resource
	}
	return base + "/" + defaultVersionPrefix + resource
}
