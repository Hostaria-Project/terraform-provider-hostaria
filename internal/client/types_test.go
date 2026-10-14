package client

import (
	"encoding/json"
	"testing"
)

func TestFlexIDUnmarshalNumberAndString(t *testing.T) {
	var key SSHKey
	if err := json.Unmarshal([]byte(`{"id":1,"name":"tf-smoke","public_key":"ssh-ed25519 AAAA"}`), &key); err != nil {
		t.Fatal(err)
	}
	if key.ID.String() != "1" {
		t.Fatalf("numeric id=%q", key.ID)
	}
	if err := json.Unmarshal([]byte(`{"id":"key_1","name":"ci","public_key":"ssh-ed25519 AAAA"}`), &key); err != nil {
		t.Fatal(err)
	}
	if key.ID.String() != "key_1" {
		t.Fatalf("string id=%q", key.ID)
	}
}
