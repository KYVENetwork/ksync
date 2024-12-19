@test "KYVE: height sync below first snapshot height" {
  run ./build/ksync height-sync --opt-out -b $HOME/bins/kyved-v1.0.0 -c kaon-1 -t 50 -r -d -y
  [ "$status" -eq 0 ]
}

@test "KYVE: try to height-sync if the app has not been resetted" {
  run ./build/ksync height-sync --opt-out -b $HOME/bins/kyved-v1.0.0 -c kaon-1 -t 100 -d -y
  [ "$status" -eq 0 ]
}

@test "KYVE Cosmovisor: try to height-sync with an upgrade between snapshot and target height" {
  run ./build/ksync height-sync --opt-out -b cosmovisor -c kaon-1 -t 2061120 -r -d -a -y
  [ "$status" -eq 0 ]
}

@test "dYdX: height sync to specific height" {
  run ./build/ksync height-sync --opt-out -b $HOME/bins/dydxprotocold-v2.0.1 -c kaon-1 -t 5935178 -r -d -y
  [ "$status" -eq 0 ]
}

@test "Archway: height sync below first snapshot height" {
  run ./build/ksync height-sync --opt-out -b $HOME/bins/archwayd-v1.0.1 -t 50 -r -d -y
  [ "$status" -eq 0 ]
}

@test "Celestia: height sync to specific height" {
  run ./build/ksync height-sync --opt-out -b $HOME/bins/celestia-appd-v1.3.0 -t 10010 -r -d -y
  [ "$status" -eq 0 ]
}

@test "Andromeda: height-sync to specific height" {
  run ./build/ksync height-sync --opt-out -b $HOME/bins/andromedad-1-v0.1.1-beta-patch -c kaon-1 -t 2700020 -r -d -y
  [ "$status" -eq 0 ]
}

@test "Noble: height-sync to specific height" {
  run ./build/ksync height-sync --opt-out -b $HOME/bins/nobled-v8.0.3 -t 16557020 -r -d -y
  [ "$status" -eq 0 ]
}

@test "Noble: height-sync to specific height on snapshot height" {
  run ./build/ksync height-sync --opt-out -b $HOME/bins/nobled-v8.0.3 -t 16554000 -r -d -y
  [ "$status" -eq 0 ]
}
