package client

import (
	"strings"
	"testing"
)

func TestJoinAPI_neverDoublePrefix(t *testing.T) {
	cases := []struct {
		endpoint, resource, want string
	}{
		{"https://api.hostaria.cloud", "/vps", "https://api.hostaria.cloud/v1/vps"},
		{"https://api.hostaria.cloud/", "vps", "https://api.hostaria.cloud/v1/vps"},
		{"https://api.hostaria.cloud/v1", "/vps", "https://api.hostaria.cloud/v1/vps"},
		{"https://api.hostaria.cloud/v1/", "/vps", "https://api.hostaria.cloud/v1/vps"},
		{"https://api.hostaria.cloud/api", "/vps", "https://api.hostaria.cloud/v1/vps"},
		{"https://api.hostaria.cloud/v1/api", "/vps", "https://api.hostaria.cloud/v1/vps"},
		{"https://api.hostaria.cloud", "/api/v1/vps", "https://api.hostaria.cloud/v1/vps"},
		{"https://api.hostaria.cloud/v1", "/api/v1/vps", "https://api.hostaria.cloud/v1/vps"},
		{"https://api.hostaria.cloud/v1", "/v1/vps", "https://api.hostaria.cloud/v1/vps"},
		{"https://api.hostaria.cloud", "/offers/1", "https://api.hostaria.cloud/v1/offers/1"},
		{"http://127.0.0.1:18080", "/os", "http://127.0.0.1:18080/v1/os"},
		{"http://127.0.0.1:18080/v1", "/game-servers", "http://127.0.0.1:18080/v1/game-servers"},
	}
	for _, tc := range cases {
		got := JoinAPI(tc.endpoint, tc.resource)
		if got != tc.want {
			t.Errorf("JoinAPI(%q, %q)=%q want %q", tc.endpoint, tc.resource, got, tc.want)
		}
		if strings.Contains(got, "/v1/api") || strings.Contains(got, "/v1/v1/") {
			t.Errorf("JoinAPI(%q, %q)=%q has stacked prefix", tc.endpoint, tc.resource, got)
		}
	}
}
