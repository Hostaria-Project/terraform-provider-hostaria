package client_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/hostaria/terraform-provider-hostaria/internal/client"
	"github.com/hostaria/terraform-provider-hostaria/internal/mockapi"
)

func TestMockAPIGarbageDoesNotCrash(t *testing.T) {
	srv := mockapi.New(t)
	base := srv.URL + "/v1"

	type probe struct {
		name   string
		method string
		path   string
		body   string
		token  string
		want   []int // allowed statuses (no 5xx)
	}
	probes := []probe{
		{"no auth", http.MethodGet, "/vps", "", "", []int{401}},
		{"empty bearer", http.MethodGet, "/vps", "", "Bearer ", []int{401}},
		{"garbage token list", http.MethodGet, "/vps", "", "Bearer merde", []int{200}}, // mock only checks non-empty
		{"not json", http.MethodPost, "/vps", "ceci n'est pas du json", "Bearer t", []int{400}},
		{"truncated json", http.MethodPost, "/vps", `{"offer_id":`, "Bearer t", []int{400}},
		{"array body", http.MethodPost, "/vps", `[]`, "Bearer t", []int{400}},
		{"null body", http.MethodPost, "/vps", `null`, "Bearer t", []int{400}},
		{"offer as string", http.MethodPost, "/vps", `{"offer_id":"nope","os_id":3}`, "Bearer t", []int{400}},
		{"missing ids", http.MethodPost, "/vps", `{"name":"<script>"}`, "Bearer t", []int{400}},
		{"unknown path", http.MethodGet, "/vps/../../etc/passwd", "", "Bearer t", []int{401, 404}},
		{"weird uuid", http.MethodGet, "/vps/%00%ff", "", "Bearer t", []int{401, 404}},
		{"ssh empty key", http.MethodPost, "/ssh-keys", `{"name":"x","public_key":""}`, "Bearer t", []int{400}},
		{"ssh not json", http.MethodPost, "/ssh-keys", `MERDE`, "Bearer t", []int{400}},
		{"game unknown", http.MethodPost, "/game-servers", `{"game":"minecraft","offer_id":1}`, "Bearer t", []int{400}},
		{"xss name still creates on mock", http.MethodPost, "/vps", `{"offer_id":1,"os_id":3,"name":"<script>alert(1)</script>"}`, "Bearer t", []int{202}},
	}

	for _, p := range probes {
		t.Run(p.name, func(t *testing.T) {
			var rdr io.Reader
			if p.body != "" {
				rdr = strings.NewReader(p.body)
			}
			req, err := http.NewRequest(p.method, base+p.path, rdr)
			if err != nil {
				t.Fatal(err)
			}
			if p.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			if p.token != "" {
				req.Header.Set("Authorization", p.token)
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("transport: %v", err)
			}
			defer resp.Body.Close()
			raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
			if resp.StatusCode >= 500 {
				t.Fatalf("5xx %d body=%s", resp.StatusCode, truncateBytes(raw, 200))
			}
			ok := false
			for _, w := range p.want {
				if resp.StatusCode == w {
					ok = true
					break
				}
			}
			if !ok {
				t.Fatalf("status %d want %v body=%s", resp.StatusCode, p.want, truncateBytes(raw, 200))
			}
		})
	}
}

func TestMockAPIHugeJSONIsRejectedOrHandled(t *testing.T) {
	srv := mockapi.New(t)
	// ~200 KiB, assez pour stresser le decode sans OOM.
	huge := `{"offer_id":1,"os_id":3,"name":"` + strings.Repeat("A", 200_000) + `"}`
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/v1/vps", strings.NewReader(huge))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer t")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 500 {
		t.Fatalf("panic/5xx on huge name: %d", resp.StatusCode)
	}
}

func TestClientGarbageAPIErrorParse(t *testing.T) {
	srv := mockapi.New(t)
	c := client.New(srv.URL, "t")
	_, err := c.CreateVM(context.Background(), client.CreateVMRequest{Name: "x"})
	if err == nil {
		t.Fatal("expected error")
	}
	ae, ok := err.(*client.APIError)
	if !ok || ae.Status != 400 {
		t.Fatalf("got %v", err)
	}
}

func TestFlexIDGarbage(t *testing.T) {
	var id client.FlexID
	bads := [][]byte{[]byte(`{}`), []byte(`true`), []byte(`[]`)}
	for _, b := range bads {
		if err := json.Unmarshal(b, &id); err == nil {
			t.Fatalf("accepted %s", b)
		}
	}
	if err := json.Unmarshal([]byte(`"abc"`), &id); err != nil || id.String() != "abc" {
		t.Fatalf("string id: %v %q", err, id)
	}
}

func truncateBytes(b []byte, n int) string {
	b = bytes.TrimSpace(b)
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "…"
}
