package provider

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hostaria/terraform-provider-hostaria/internal/mockapi"
)

func testAccProtoV6ProviderFactories() map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"hostaria": providerserver.NewProtocol6WithError(New("test")()),
	}
}

func TestAccHostariaResources(t *testing.T) {
	srv := mockapi.New(t)
	cfg := fmt.Sprintf(`
provider "hostaria" {
  endpoint = %q
  api_key  = "test-token"
}

data "hostaria_offers" "all" {}

data "hostaria_offer" "starter" {
  name = "starter"
}

data "hostaria_os" "ubuntu" {
  slug = "ubuntu-24-04"
}

resource "hostaria_ssh_key" "ci" {
  name       = "ci"
  public_key = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA ci@hostaria"
}

resource "hostaria_vps" "web" {
  offer_id = data.hostaria_offer.starter.id
  os_id    = data.hostaria_os.ubuntu.id
  name     = "prod-web-01"

  bootstrap {
    username    = "deploy"
    auth_method = "ssh_only"
    ssh_key_ids = [hostaria_ssh_key.ci.id]
  }
}

resource "hostaria_game_server" "ark" {
  game     = "ark"
  offer_id = 2
  name     = "my-ark-server"
}
`, srv.URL)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: cfg,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.hostaria_offer.starter", "name", "starter"),
					resource.TestCheckResourceAttr("data.hostaria_os.ubuntu", "slug", "ubuntu-24-04"),
					resource.TestCheckResourceAttr("hostaria_vps.web", "name", "prod-web-01"),
					resource.TestCheckResourceAttr("hostaria_vps.web", "status", "running"),
					resource.TestCheckResourceAttrSet("hostaria_vps.web", "ip_address"),
					resource.TestCheckResourceAttrSet("hostaria_vps.web", "uuid"),
					resource.TestCheckResourceAttrSet("hostaria_vps.web", "hostname"),
					resource.TestCheckResourceAttr("hostaria_ssh_key.ci", "name", "ci"),
					resource.TestCheckResourceAttr("hostaria_game_server.ark", "game", "ark"),
					resource.TestCheckResourceAttr("hostaria_game_server.ark", "status", "running"),
				),
			},
			{
				Config: strings.Replace(cfg, `name     = "prod-web-01"`, `name     = "prod-web-01"
  cpu_cores = 3`, 1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("hostaria_vps.web", "cpu_cores", "3"),
				),
			},
			{
				ResourceName:            "hostaria_vps.web",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"bootstrap"},
			},
		},
	})
}
