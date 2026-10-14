package client_test

import (
	"context"
	"testing"

	"github.com/hostaria/terraform-provider-hostaria/internal/client"
	"github.com/hostaria/terraform-provider-hostaria/internal/mockapi"
)

func TestClientCRUDAgainstMock(t *testing.T) {
	srv := mockapi.New(t)
	c := client.New(srv.URL, "test-token")
	ctx := context.Background()

	offers, err := c.ListOffers(ctx)
	if err != nil || len(offers) != 2 {
		t.Fatalf("ListOffers: %v %#v", err, offers)
	}
	off, err := c.GetOffer(ctx, 1)
	if err != nil || off.Name != "starter" {
		t.Fatalf("GetOffer: %v %#v", err, off)
	}

	oses, err := c.ListOS(ctx)
	if err != nil || len(oses) != 2 {
		t.Fatalf("ListOS: %v %#v", err, oses)
	}

	vm, err := c.CreateVM(ctx, client.CreateVMRequest{
		Name:    "prod-web-01",
		OfferID: 1,
		OsID:    3,
		Bootstrap: &client.Bootstrap{
			Username:      "deploy",
			AuthMethod:    "ssh_only",
			SSHPublicKeys: []string{"ssh-ed25519 AAAA test"},
		},
	})
	if err != nil {
		t.Fatalf("CreateVM: %v", err)
	}
	if vm.ID == 0 || vm.IDVm == "" || vm.Name != "prod-web-01" {
		t.Fatalf("CreateVM result: %#v", vm)
	}
	if vm.VmUsername == nil || *vm.VmUsername != "deploy" {
		t.Fatalf("username: %#v", vm.VmUsername)
	}

	got, err := c.GetVM(ctx, vm.IDVm)
	if err != nil || got.ID != vm.ID {
		t.Fatalf("GetVM uuid: %v %#v", err, got)
	}
	got, err = c.GetVM(ctx, "1")
	if err != nil || got.ID != 1 {
		t.Fatalf("GetVM numeric: %v %#v", err, got)
	}

	if err := c.DeleteVM(ctx, vm.IDVm); err != nil {
		t.Fatalf("DeleteVM: %v", err)
	}
	if _, err := c.GetVM(ctx, vm.IDVm); !client.IsNotFound(err) {
		t.Fatalf("expected 404 after delete, got %v", err)
	}

	key, err := c.CreateSSHKey(ctx, client.SSHKey{Name: "ci", PublicKey: "ssh-ed25519 AAAA ci"})
	if err != nil || key.ID == "" {
		t.Fatalf("CreateSSHKey: %v %#v", err, key)
	}
	if err := c.DeleteSSHKey(ctx, key.ID.String()); err != nil {
		t.Fatalf("DeleteSSHKey: %v", err)
	}

	gs, err := c.CreateGameServer(ctx, client.GameServer{Game: "ark", Name: "my-ark", OfferID: 2})
	if err != nil || gs.ID == "" {
		t.Fatalf("CreateGameServer: %v %#v", err, gs)
	}
	if err := c.DeleteGameServer(ctx, gs.ID); err != nil {
		t.Fatalf("DeleteGameServer: %v", err)
	}
}

func TestClientUnwrapsEnvelope(t *testing.T) {
	srv := mockapi.New(t)
	srv.UseEnvelope(true)
	c := client.New(srv.URL+"/v1", "tok")
	offers, err := c.ListOffers(context.Background())
	if err != nil || len(offers) != 2 {
		t.Fatalf("enveloped ListOffers: %v %#v", err, offers)
	}
}

func TestClientUnauthorized(t *testing.T) {
	srv := mockapi.New(t)
	c := client.New(srv.URL, "")
	_, err := c.ListOS(context.Background())
	if err == nil {
		t.Fatal("expected 401")
	}
	ae, ok := err.(*client.APIError)
	if !ok || ae.Status != 401 {
		t.Fatalf("got %v", err)
	}
}
