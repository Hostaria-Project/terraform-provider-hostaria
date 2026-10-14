data "hostaria_offer" "starter" {
  name = "Starter"
}

data "hostaria_os" "ubuntu" {
  slug = "ubuntu-24-04"
}

resource "hostaria_vps" "web" {
  offer_id = data.hostaria_offer.starter.id
  os_id    = data.hostaria_os.ubuntu.id
  name     = "web-01"

  bootstrap {
    username        = "deploy"
    auth_method     = "ssh_only"
    ssh_public_keys = ["ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA ci@hostaria"]
  }
}
