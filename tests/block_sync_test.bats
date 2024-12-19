@test "KYVE: block sync 50 blocks from genesis" {
  run ./build/ksync block-sync --opt-out -b $HOME/bins/kyved-v1.0.0 -c kaon-1 -t 50 -r -d -y
  [ "$status" -eq 0 ]
}

@test "KYVE: continue block sync from height 50" {
  run ./build/ksync block-sync --opt-out -b $HOME/bins/kyved-v1.0.0 -c kaon-1 -t 100 -d -y
  [ "$status" -eq 0 ]
}

@test "KYVE: try to block-sync with target height lower than current one" {
  run ./build/ksync block-sync --opt-out -b $HOME/bins/kyved-v1.0.0 -c kaon-1 -t 50 -d -y
  [ "$status" -eq 1 ]
}

@test "dYdX: block sync 50 blocks from genesis" {
  run ./build/ksync block-sync --opt-out -b $HOME/bins/dydxprotocold-v2.0.1 -t 50 -r -d -y
  [ "$status" -eq 0 ]
}

@test "dYdX: continue block sync from height 50" {
  run ./build/ksync block-sync --opt-out -b $HOME/bins/dydxprotocold-v2.0.1 -t 100 -d -y
  [ "$status" -eq 0 ]
}

@test "Archway: block sync 50 blocks from genesis with p2p bootstrap" {
  run ./build/ksync block-sync --opt-out -b $HOME/bins/archwayd-v1.0.1 -t 50 -r -d -y
  [ "$status" -eq 0 ]
}

@test "Archway: continue block sync from height 50" {
  run ./build/ksync block-sync --opt-out -b $HOME/bins/archwayd-v1.0.1 -t 100 -d -y
  [ "$status" -eq 0 ]
}

@test "Celestia: block sync 10 blocks from genesis" {
  run ./build/ksync block-sync --opt-out -b $HOME/bins/celestia-appd-v1.3.0 -t 10 -r -d -y
  [ "$status" -eq 0 ]
}
