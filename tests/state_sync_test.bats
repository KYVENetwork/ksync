@test "KYVE: state-sync exact height" {
  run ./build/ksync state-sync --opt-out -b $HOME/bins/kyved-v1.0.0 -c kaon-1 -t 12000 -r -d -y
  [ "$status" -eq 0 ]
}

@test "KYVE: state-sync recommended nearest height" {
  run ./build/ksync state-sync --opt-out -b $HOME/bins/kyved-v1.0.0 -c kaon-1 -t 15302 -r -d -y
  [ "$status" -eq 0 ]
}

@test "KYVE: try to state-sync if the app has not been resetted" {
  run ./build/ksync state-sync --opt-out -b $HOME/bins/kyved-v1.0.0 -c kaon-1 -t 12000 -d -y
  [ "$status" -eq 1 ]
}

@test "dYdX: state-sync exact height" {
  run ./build/ksync state-sync --opt-out -b $HOME/bins/dydxprotocold-v2.0.1 -c kaon-1 -t 500000 -r -d -y
  [ "$status" -eq 0 ]
}

@test "Celestia: state-sync exact height" {
  run ./build/ksync state-sync --opt-out -b $HOME/bins/celestia-appd-v1.3.0 -t 10000 -r -d -y
  [ "$status" -eq 0 ]
}
