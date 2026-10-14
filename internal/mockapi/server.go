package mockapi

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/hostaria/terraform-provider-hostaria/internal/client"
)

type Server struct {
	*httptest.Server
	mu           sync.Mutex
	offers       []client.Offer
	oses         []client.OS
	vms          map[int64]*client.VM
	sshKeys      map[string]*client.SSHKey
	gameServers  map[string]*client.GameServer
	nextVM       int64
	nextKey      int64
	nextGame     int64
	envelope     bool
	requireToken bool
}

func NewServer() *Server {
	return &Server{
		vms:          map[int64]*client.VM{},
		sshKeys:      map[string]*client.SSHKey{},
		gameServers:  map[string]*client.GameServer{},
		nextVM:       1,
		nextKey:      1,
		nextGame:     1,
		requireToken: true,
		offers: []client.Offer{
			{ID: 1, Name: "starter", Price: 5.99, Ram: 2048, CpuCores: 2, Storage: 40, BillingInterval: "month", IsPublic: true},
			{ID: 2, Name: "game-standard", Price: 19.99, Ram: 8192, CpuCores: 4, Storage: 80, BillingInterval: "month", IsPublic: true},
		},
		oses: []client.OS{
			{ID: 3, Name: "Ubuntu 24.04", Slug: "ubuntu-24-04", OS: "ubuntu", Version: "24.04", Family: "linux", IsActive: true},
			{ID: 4, Name: "Debian 12", Slug: "debian-12", OS: "debian", Version: "12", Family: "linux", IsActive: true},
		},
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.route)
	return mux
}

func New(t testing.TB) *Server {
	t.Helper()
	s := NewServer()
	s.Server = httptest.NewServer(s.Handler())
	t.Cleanup(s.Close)
	return s
}

func (s *Server) UseEnvelope(v bool) { s.envelope = v }

func (s *Server) route(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if strings.HasPrefix(path, "/v1/") {
		path = strings.TrimPrefix(path, "/v1")
	} else if path == "/v1" {
		path = "/"
	}

	if path != "/offers" && !strings.HasPrefix(path, "/offers/") {
		if s.requireToken {
			auth := r.Header.Get("Authorization")
			if !strings.HasPrefix(auth, "Bearer ") || strings.TrimSpace(strings.TrimPrefix(auth, "Bearer ")) == "" {
				s.writeErr(w, http.StatusUnauthorized, "Unauthorized")
				return
			}
		}
	}

	switch {
	case r.Method == http.MethodGet && path == "/offers":
		s.writeJSON(w, http.StatusOK, s.offers)
	case r.Method == http.MethodGet && strings.HasPrefix(path, "/offers/"):
		id, _ := strconv.ParseInt(strings.TrimPrefix(path, "/offers/"), 10, 64)
		s.mu.Lock()
		defer s.mu.Unlock()
		for _, o := range s.offers {
			if o.ID == id {
				s.writeJSON(w, http.StatusOK, o)
				return
			}
		}
		s.writeErr(w, http.StatusNotFound, "Offer not found")
	case r.Method == http.MethodGet && path == "/os":
		s.writeJSON(w, http.StatusOK, s.oses)
	case r.Method == http.MethodGet && strings.HasPrefix(path, "/os/"):
		id, _ := strconv.ParseInt(strings.TrimPrefix(path, "/os/"), 10, 64)
		s.mu.Lock()
		defer s.mu.Unlock()
		for _, o := range s.oses {
			if o.ID == id {
				s.writeJSON(w, http.StatusOK, o)
				return
			}
		}
		s.writeErr(w, http.StatusNotFound, "OS not found")
	case strings.HasPrefix(path, "/vps"):
		s.handleVPS(w, r, path)
	case strings.HasPrefix(path, "/ssh-keys"):
		s.handleSSH(w, r, path)
	case strings.HasPrefix(path, "/game-servers"):
		s.handleGames(w, r, path)
	default:
		s.writeErr(w, http.StatusNotFound, "Not found")
	}
}

func (s *Server) handleVPS(w http.ResponseWriter, r *http.Request, path string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rest := strings.TrimPrefix(path, "/vps")
	rest = strings.TrimPrefix(rest, "/")

	switch {
	case r.Method == http.MethodGet && rest == "":
		list := make([]client.VM, 0, len(s.vms))
		for _, vm := range s.vms {
			list = append(list, *vm)
		}
		s.writeJSON(w, http.StatusOK, list)
	case r.Method == http.MethodPost && rest == "":
		var req client.CreateVMRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.writeErr(w, http.StatusBadRequest, "Invalid request body")
			return
		}
		if req.OfferID == 0 || req.OsID == 0 {
			s.writeErr(w, http.StatusBadRequest, "offer_id and os_id are required")
			return
		}
		id := s.nextVM
		s.nextVM++
		uuid := "aaaaaaaa-bbbb-cccc-dddd-" + pad12(id)
		ip := "10.10.20." + strconv.FormatInt(10+id, 10)
		name := req.Name
		if name == "" {
			name = "vps-" + strconv.FormatInt(id, 10)
		}
		user := req.VmUsername
		if user == "" && req.Bootstrap != nil {
			user = req.Bootstrap.Username
		}
		if user == "" {
			user = "admin"
		}
		cpu, ram, disk := s.offerSpecs(req.OfferID)
		if req.CpuCores > 0 {
			cpu = req.CpuCores
		}
		if req.RamMB > 0 {
			ram = req.RamMB
		}
		if req.StorageGB > 0 {
			if req.StorageGB < disk {
				s.writeErr(w, http.StatusBadRequest, "Le disque ne peut qu'être agrandi.")
				return
			}
			disk = req.StorageGB
		}
		vm := &client.VM{
			ID:         id,
			OfferID:    req.OfferID,
			OsID:       req.OsID,
			Name:       name,
			IDVm:       uuid,
			Status:     "running",
			IpAddress:  &ip,
			VmUsername: &user,
			OwnerID:    1,
			CpuCores:   cpu,
			RamMB:      ram,
			StorageGB:  disk,
		}
		s.vms[id] = vm
		s.writeJSON(w, http.StatusAccepted, map[string]any{
			"status":     "pending",
			"id":         id,
			"vm_uuid":    uuid,
			"id_vm":      uuid,
			"vm_name":    name,
			"ip_address": ip,
		})
	case rest != "":
		idPart := rest
		if i := strings.IndexByte(rest, '/'); i >= 0 {
			idPart = rest[:i]
		}
		vm := s.findVM(idPart)
		if vm == nil {
			s.writeErr(w, http.StatusNotFound, "VM not found")
			return
		}
		switch r.Method {
		case http.MethodGet:
			s.writeJSON(w, http.StatusOK, vm)
		case http.MethodPut:
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if st, ok := body["status"].(string); ok {
				vm.Status = st
			}
			if v, ok := jsonInt64(body["offer_id"]); ok {
				_, ram, disk := s.offerSpecs(v)
				if disk < vm.StorageGB {
					s.writeErrCode(w, http.StatusBadRequest, "disk_down", "Le disque ne peut qu'être agrandi.")
					return
				}
				vm.OfferID = v
				if vm.RamMB < ram {
					vm.RamMB = ram
				}
				cpu, _, _ := s.offerSpecs(v)
				if vm.CpuCores < cpu {
					vm.CpuCores = cpu
				}
			}
			if v, ok := jsonInt64(body["cpu_cores"]); ok {
				vm.CpuCores = v
			}
			if v, ok := jsonInt64(body["ram_mb"]); ok {
				vm.RamMB = v
			}
			if v, ok := jsonInt64(body["storage_gb"]); ok {
				if v < vm.StorageGB {
					s.writeErrCode(w, http.StatusBadRequest, "disk_down", "Le disque ne peut qu'être agrandi.")
					return
				}
				vm.StorageGB = v
			}
			s.writeJSON(w, http.StatusOK, vm)
		case http.MethodDelete:
			delete(s.vms, vm.ID)
			w.WriteHeader(http.StatusNoContent)
		default:
			s.writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	}
}

func (s *Server) findVM(id string) *client.VM {
	if n, err := strconv.ParseInt(id, 10, 64); err == nil {
		return s.vms[n]
	}
	for _, vm := range s.vms {
		if vm.IDVm == id {
			return vm
		}
	}
	return nil
}

func (s *Server) handleSSH(w http.ResponseWriter, r *http.Request, path string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rest := strings.Trim(strings.TrimPrefix(path, "/ssh-keys"), "/")
	switch {
	case r.Method == http.MethodGet && rest == "":
		list := make([]client.SSHKey, 0, len(s.sshKeys))
		for _, k := range s.sshKeys {
			list = append(list, *k)
		}
		s.writeJSON(w, http.StatusOK, list)
	case r.Method == http.MethodPost && rest == "":
		var key client.SSHKey
		if err := json.NewDecoder(r.Body).Decode(&key); err != nil {
			s.writeErr(w, http.StatusBadRequest, "Invalid request body")
			return
		}
		if strings.TrimSpace(key.PublicKey) == "" {
			s.writeErr(w, http.StatusBadRequest, "public_key is required")
			return
		}
		id := "key_" + strconv.FormatInt(s.nextKey, 10)
		s.nextKey++
		if key.Name == "" {
			key.Name = id
		}
		key.ID = client.FlexID(id)
		s.sshKeys[id] = &key
		s.writeJSON(w, http.StatusCreated, key)
	case rest != "":
		key := s.sshKeys[rest]
		if key == nil {
			s.writeErr(w, http.StatusNotFound, "SSH key not found")
			return
		}
		switch r.Method {
		case http.MethodGet:
			s.writeJSON(w, http.StatusOK, key)
		case http.MethodDelete:
			delete(s.sshKeys, rest)
			w.WriteHeader(http.StatusNoContent)
		default:
			s.writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	}
}

func (s *Server) handleGames(w http.ResponseWriter, r *http.Request, path string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rest := strings.Trim(strings.TrimPrefix(path, "/game-servers"), "/")
	switch {
	case r.Method == http.MethodGet && rest == "":
		list := make([]client.GameServer, 0, len(s.gameServers))
		for _, g := range s.gameServers {
			list = append(list, *g)
		}
		s.writeJSON(w, http.StatusOK, list)
	case r.Method == http.MethodPost && rest == "":
		var gs client.GameServer
		if err := json.NewDecoder(r.Body).Decode(&gs); err != nil {
			s.writeErr(w, http.StatusBadRequest, "Invalid request body")
			return
		}
		if gs.Game != "ark" && gs.Game != "satisfactory" {
			s.writeErr(w, http.StatusBadRequest, "game must be ark or satisfactory")
			return
		}
		id := "gs_" + strconv.FormatInt(s.nextGame, 10)
		s.nextGame++
		gs.ID = id
		if gs.Status == "" {
			gs.Status = "running"
		}
		s.gameServers[id] = &gs
		s.writeJSON(w, http.StatusCreated, gs)
	case rest != "":
		gs := s.gameServers[rest]
		if gs == nil {
			s.writeErr(w, http.StatusNotFound, "game server not found")
			return
		}
		switch r.Method {
		case http.MethodGet:
			s.writeJSON(w, http.StatusOK, gs)
		case http.MethodPut:
			body, _ := io.ReadAll(r.Body)
			var upd client.GameServer
			if err := json.Unmarshal(body, &upd); err != nil {
				s.writeErr(w, http.StatusBadRequest, "Invalid request body")
				return
			}
			if upd.Name != "" {
				gs.Name = upd.Name
			}
			s.writeJSON(w, http.StatusOK, gs)
		case http.MethodDelete:
			delete(s.gameServers, rest)
			w.WriteHeader(http.StatusNoContent)
		default:
			s.writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	}
}

func (s *Server) writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if s.envelope {
		_ = json.NewEncoder(w).Encode(map[string]any{"data": v})
		return
	}
	_ = json.NewEncoder(w).Encode(v)
}

func (s *Server) writeErr(w http.ResponseWriter, status int, msg string) {
	s.writeErrCode(w, status, "", msg)
}

func (s *Server) writeErrCode(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if s.envelope {
		errObj := map[string]string{"message": msg}
		if code != "" {
			errObj["code"] = code
		} else {
			errObj["code"] = http.StatusText(status)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"error": errObj})
		return
	}
	body := map[string]string{"error": msg}
	if code != "" {
		body["code"] = code
	}
	_ = json.NewEncoder(w).Encode(body)
}

func (s *Server) offerSpecs(id int64) (cpu, ram, disk int64) {
	for _, o := range s.offers {
		if o.ID == id {
			return o.CpuCores, o.Ram, o.Storage
		}
	}
	return 2, 2048, 40
}

func jsonInt64(v any) (int64, bool) {
	switch n := v.(type) {
	case float64:
		return int64(n), true
	case int64:
		return n, true
	case int:
		return int64(n), true
	case json.Number:
		i, err := n.Int64()
		return i, err == nil
	case string:
		i, err := strconv.ParseInt(n, 10, 64)
		return i, err == nil
	default:
		return 0, false
	}
}

func pad12(id int64) string {
	s := strconv.FormatInt(id, 10)
	return strings.Repeat("0", 12-len(s)) + s
}
