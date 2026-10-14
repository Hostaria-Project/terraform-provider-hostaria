resource "hostaria_ssh_key" "ci" {
  name       = "ci"
  public_key = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA ci@hostaria"
}
