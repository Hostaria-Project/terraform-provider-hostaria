data "hostaria_offer" "starter" {
  name = "Starter"
}

resource "hostaria_game_server" "ark" {
  game     = "ark"
  name     = "my-ark-server"
  offer_id = data.hostaria_offer.starter.id
}
