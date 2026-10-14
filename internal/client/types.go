package client

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type APIError struct {
	Status  int
	Code    string
	Message string
}

func (e *APIError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("hostaria api: %d %s: %s", e.Status, e.Code, e.Message)
	}
	return fmt.Sprintf("hostaria api: %d: %s", e.Status, e.Message)
}

func IsNotFound(err error) bool {
	ae, ok := err.(*APIError)
	return ok && ae.Status == 404
}

type Offer struct {
	ID              int64   `json:"id"`
	Name            string  `json:"name"`
	Description     *string `json:"description"`
	Price           float64 `json:"price"`
	Ram             int64   `json:"ram"`
	CpuCores        int64   `json:"cpu_cores"`
	Storage         int64   `json:"storage"`
	StorageType     *string `json:"storage_type"`
	BillingInterval string  `json:"billing_interval"`
	IsPublic        bool    `json:"is_public"`
}

type OS struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	OS       string `json:"os"`
	Version  string `json:"version"`
	Family   string `json:"family"`
	IsActive bool   `json:"is_active"`
}

type VM struct {
	ID             int64           `json:"id"`
	OfferID        int64           `json:"offer_id"`
	OsID           int64           `json:"os_id"`
	Name           string          `json:"name"`
	IDVm           string          `json:"id_vm"`
	Status         string          `json:"status"`
	IpAddress      *string         `json:"ip_address"`
	VmUsername     *string         `json:"vm_username"`
	OwnerID        int64           `json:"owner_id"`
	Until          *string         `json:"metered_until,omitempty"`
	BillingCycle   string          `json:"billing_cycle,omitempty"`
	CpuCores       int64           `json:"cpu_cores,omitempty"`
	RamMB          int64           `json:"ram_mb,omitempty"`
	StorageGB      int64           `json:"storage_gb,omitempty"`
	SuggestedOffer *SuggestedOffer `json:"suggested_offer,omitempty"`
}

type SuggestedOffer struct {
	ID                int64  `json:"id"`
	Name              string `json:"name"`
	CpuCores          int64  `json:"cpu_cores,omitempty"`
	RamMB             int64  `json:"ram_mb,omitempty"`
	StorageGB         int64  `json:"storage_gb,omitempty"`
	MonthlyPriceCents int64  `json:"monthly_price_cents,omitempty"`
	SavingsCents      int64  `json:"savings_cents,omitempty"`
}

type CreateVMRequest struct {
	Name       string     `json:"name"`
	OfferID    int64      `json:"offer_id"`
	OsID       int64      `json:"os_id"`
	VmUsername string     `json:"vm_username,omitempty"`
	VmPassword string     `json:"vm_password,omitempty"`
	SshKeys    []string   `json:"ssh_keys,omitempty"`
	Until      string     `json:"until,omitempty"`
	Bootstrap  *Bootstrap `json:"bootstrap,omitempty"`
	CpuCores   int64      `json:"cpu_cores,omitempty"`
	RamMB      int64      `json:"ram_mb,omitempty"`
	StorageGB  int64      `json:"storage_gb,omitempty"`
}

type Bootstrap struct {
	Username       string   `json:"username,omitempty"`
	AuthMethod     string   `json:"auth_method,omitempty"`
	SSHPublicKeys  []string `json:"ssh_public_keys,omitempty"`
	SSHKeyIDs      []string `json:"ssh_key_ids,omitempty"`
	Password       string   `json:"password,omitempty"`
	UserData       string   `json:"user_data,omitempty"`
	UserDataFormat string   `json:"user_data_format,omitempty"`
}

type createVMResponse struct {
	ID        int64  `json:"id"`
	Status    string `json:"status"`
	VmUUID    string `json:"vm_uuid"`
	IDVm      string `json:"id_vm"`
	Name      string `json:"name"`
	VmName    string `json:"vm_name"`
	IpAddress string `json:"ip_address"`
}

// FlexID accepte un id JSON nombre (API prod) ou string (mock).
type FlexID string

func (id FlexID) String() string { return string(id) }

func (id FlexID) MarshalJSON() ([]byte, error) {
	s := string(id)
	if s == "" {
		return []byte("null"), nil
	}
	if _, err := strconv.ParseInt(s, 10, 64); err == nil {
		return []byte(s), nil
	}
	return json.Marshal(s)
}

func (id *FlexID) UnmarshalJSON(b []byte) error {
	s := strings.TrimSpace(string(b))
	if s == "" || s == "null" {
		*id = ""
		return nil
	}
	if len(s) > 0 && s[0] == '"' {
		var v string
		if err := json.Unmarshal(b, &v); err != nil {
			return err
		}
		*id = FlexID(v)
		return nil
	}
	var n json.Number
	if err := json.Unmarshal(b, &n); err != nil {
		return fmt.Errorf("id: attendu string ou number, reçu %s", s)
	}
	*id = FlexID(n.String())
	return nil
}

type SSHKey struct {
	ID          FlexID `json:"id,omitempty"`
	Name        string `json:"name"`
	PublicKey   string `json:"public_key"`
	Fingerprint string `json:"fingerprint,omitempty"`
}

type GameServer struct {
	ID      string `json:"id"`
	Game    string `json:"game"`
	Name    string `json:"name"`
	OfferID int64  `json:"offer_id"`
	Status  string `json:"status"`
}
