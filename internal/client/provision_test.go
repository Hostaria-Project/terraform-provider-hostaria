package client

import "testing"

func TestProvisionFailure(t *testing.T) {
	if got := provisionFailure(&VM{Status: "running"}); got != "" {
		t.Fatalf("running: %q", got)
	}
	if got := provisionFailure(&VM{Status: "creating"}); got != "" {
		t.Fatalf("creating: %q", got)
	}
	want := "Le système d'exploitation choisi n'est pas disponible. Choisissez un autre OS."
	if got := provisionFailure(&VM{Status: "error", Error: want}); got != want {
		t.Fatalf("error: %q", got)
	}
	if got := provisionFailure(&VM{Status: "missing"}); got == "" {
		t.Fatal("missing was treated as success")
	}
}
