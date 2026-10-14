package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	http      *http.Client
	endpoint  string
	apiKey    string
	userAgent string
}

func New(endpoint, apiKey string) *Client {
	ep := strings.TrimSpace(endpoint)
	if ep == "" {
		ep = defaultEndpoint
	}
	return &Client{
		http: &http.Client{
			Timeout: 3 * time.Minute,
		},
		endpoint:  strings.TrimRight(ep, "/"),
		apiKey:    strings.TrimSpace(apiKey),
		userAgent: "terraform-provider-hostaria/dev",
	}
}

func (c *Client) do(ctx context.Context, method, resource string, body any, dest any) error {
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rdr = bytes.NewReader(b)
	}
	url := JoinAPI(c.endpoint, resource)
	req, err := http.NewRequestWithContext(ctx, method, url, rdr)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode == http.StatusNoContent || len(bytes.TrimSpace(raw)) == 0 {
		if resp.StatusCode >= 400 {
			return &APIError{Status: resp.StatusCode, Message: resp.Status}
		}
		return nil
	}

	if resp.StatusCode >= 400 {
		return parseAPIError(resp.StatusCode, raw)
	}
	if dest == nil {
		return nil
	}
	payload, err := unwrapData(raw)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(payload, dest); err != nil {
		return fmt.Errorf("decode %s %s: %w (body %s)", method, url, err, truncate(raw, 256))
	}
	return nil
}

func unwrapData(raw []byte) (json.RawMessage, error) {
	trim := bytes.TrimSpace(raw)
	if len(trim) == 0 {
		return nil, fmt.Errorf("empty response")
	}
	if trim[0] == '[' {
		return json.RawMessage(trim), nil
	}
	if trim[0] != '{' {
		return json.RawMessage(trim), nil
	}
	var env struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(trim, &env); err != nil {
		return nil, err
	}
	if len(env.Data) > 0 && string(env.Data) != "null" {
		return env.Data, nil
	}
	return json.RawMessage(trim), nil
}

func parseAPIError(status int, raw []byte) error {
	ae := &APIError{Status: status, Message: strings.TrimSpace(string(raw))}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return ae
	}
	if errRaw, ok := obj["error"]; ok {
		var msg string
		if json.Unmarshal(errRaw, &msg) == nil && msg != "" {
			ae.Message = msg
		} else {
			var nested struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			}
			if json.Unmarshal(errRaw, &nested) == nil && (nested.Message != "" || nested.Code != "") {
				ae.Code = nested.Code
				if nested.Message != "" {
					ae.Message = nested.Message
				}
			}
		}
	}
	if ae.Code == "" {
		if codeRaw, ok := obj["code"]; ok {
			var code string
			if json.Unmarshal(codeRaw, &code) == nil {
				ae.Code = code
			}
		}
	}
	return ae
}

func truncate(b []byte, n int) string {
	s := string(b)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func (c *Client) ListOffers(ctx context.Context) ([]Offer, error) {
	var out []Offer
	if err := c.do(ctx, http.MethodGet, "/offers", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) GetOffer(ctx context.Context, id int64) (*Offer, error) {
	var out Offer
	if err := c.do(ctx, http.MethodGet, "/offers/"+strconv.FormatInt(id, 10), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ListOS(ctx context.Context) ([]OS, error) {
	var out []OS
	if err := c.do(ctx, http.MethodGet, "/os", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) GetOS(ctx context.Context, id int64) (*OS, error) {
	var out OS
	if err := c.do(ctx, http.MethodGet, "/os/"+strconv.FormatInt(id, 10), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) CreateVM(ctx context.Context, req CreateVMRequest) (*VM, error) {
	if req.Bootstrap != nil {
		if req.VmUsername == "" {
			req.VmUsername = req.Bootstrap.Username
		}
		if req.VmPassword == "" {
			req.VmPassword = req.Bootstrap.Password
		}
		if len(req.SshKeys) == 0 {
			req.SshKeys = req.Bootstrap.SSHPublicKeys
		}
	}
	var created createVMResponse
	if err := c.do(ctx, http.MethodPost, "/vps", req, &created); err != nil {
		return nil, err
	}
	id := created.ID
	lookup := created.IDVm
	if lookup == "" {
		lookup = created.VmUUID
	}
	if id > 0 {
		lookup = strconv.FormatInt(id, 10)
	}
	if lookup == "" {
		return nil, fmt.Errorf("create vps: response missing id and uuid")
	}
	return c.waitVMReady(ctx, lookup)
}

func provisionFailure(vm *VM) string {
	if vm == nil {
		return ""
	}
	switch strings.ToLower(strings.TrimSpace(vm.Status)) {
	case "error", "failed", "missing":
		if msg := strings.TrimSpace(vm.Error); msg != "" {
			return msg
		}
		return "La création du VPS a échoué. Réessayez dans quelques minutes."
	default:
		return ""
	}
}

func (c *Client) waitVMReady(ctx context.Context, lookup string) (*VM, error) {
	deadline := time.Now().Add(10 * time.Minute)
	var last *VM
	for time.Now().Before(deadline) {
		vm, err := c.GetVM(ctx, lookup)
		if err != nil {
			if last != nil && IsNotFound(err) {
				return last, fmt.Errorf("vps disappeared during provision: %w", err)
			}
			if !IsNotFound(err) {
				return nil, err
			}
		} else {
			last = vm
			if msg := provisionFailure(vm); msg != "" {
				return vm, fmt.Errorf("%s", msg)
			}
			st := strings.ToLower(strings.TrimSpace(vm.Status))
			switch st {
			case "creating", "pending", "":
			default:
				return vm, nil
			}
		}
		select {
		case <-ctx.Done():
			return last, ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
	if last != nil {
		return last, fmt.Errorf("timeout waiting for vps %s (last status=%s)", lookup, last.Status)
	}
	return nil, fmt.Errorf("timeout waiting for vps %s", lookup)
}

func (c *Client) GetVM(ctx context.Context, id string) (*VM, error) {
	var out VM
	if err := c.do(ctx, http.MethodGet, "/vps/"+id, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateVM(ctx context.Context, id string, body map[string]any) (*VM, error) {
	var out VM
	if err := c.do(ctx, http.MethodPut, "/vps/"+id, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteVM(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/vps/"+id, nil, nil)
}

func (c *Client) CreateSSHKey(ctx context.Context, key SSHKey) (*SSHKey, error) {
	var out SSHKey
	if err := c.do(ctx, http.MethodPost, "/ssh-keys", key, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetSSHKey(ctx context.Context, id string) (*SSHKey, error) {
	var out SSHKey
	if err := c.do(ctx, http.MethodGet, "/ssh-keys/"+id, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteSSHKey(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/ssh-keys/"+id, nil, nil)
}

func (c *Client) CreateGameServer(ctx context.Context, gs GameServer) (*GameServer, error) {
	var out GameServer
	if err := c.do(ctx, http.MethodPost, "/game-servers", gs, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetGameServer(ctx context.Context, id string) (*GameServer, error) {
	var out GameServer
	if err := c.do(ctx, http.MethodGet, "/game-servers/"+id, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateGameServer(ctx context.Context, id string, gs GameServer) (*GameServer, error) {
	var out GameServer
	if err := c.do(ctx, http.MethodPut, "/game-servers/"+id, gs, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteGameServer(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/game-servers/"+id, nil, nil)
}

func VMLookupID(vm *VM) string {
	if vm == nil {
		return ""
	}
	if vm.IDVm != "" {
		return vm.IDVm
	}
	if vm.ID > 0 {
		return strconv.FormatInt(vm.ID, 10)
	}
	return ""
}
