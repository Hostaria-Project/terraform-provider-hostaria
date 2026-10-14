package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hostaria/terraform-provider-hostaria/internal/mockapi"
)

func TestAccGarbageConfigsAreRejected(t *testing.T) {
	srv := mockapi.New(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
provider "hostaria" {
  endpoint = %q
  api_key  = "test-token"
}
resource "hostaria_vps" "bad" {
  offer_id = "merde"
  os_id    = 3
}
`, srv.URL),
				ExpectError: regexp.MustCompile(`number|int64|offer_id`),
			},
		},
	})

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
provider "hostaria" {
  endpoint = %q
  api_key  = "test-token"
}
resource "hostaria_ssh_key" "bad" {
  name       = "<script>alert(1)</script>"
  public_key = ""
}
`, srv.URL),
				ExpectError: regexp.MustCompile(`public_key|400|required`),
			},
		},
	})

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
provider "hostaria" {
  endpoint = %q
  api_key  = "test-token"
}
resource "hostaria_game_server" "bad" {
  game     = "minecraft"
  offer_id = 1
  name     = "nope"
}
`, srv.URL),
				ExpectError: regexp.MustCompile(`ark|satisfactory|400`),
			},
		},
	})
}
