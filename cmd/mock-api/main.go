package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/hostaria/terraform-provider-hostaria/internal/mockapi"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:18080", "listen address")
	flag.Parse()

	s := mockapi.NewServer()
	fmt.Fprintf(os.Stderr, "mock Hostaria API on http://%s/v1 (token: any Bearer)\n", *addr)
	fmt.Fprintf(os.Stderr, "  GET  /v1/offers  /v1/os  /v1/vps  /v1/ssh-keys  /v1/game-servers\n")
	log.Fatal(http.ListenAndServe(*addr, s.Handler()))
}
