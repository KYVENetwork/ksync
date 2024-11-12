@test "KYVE: height sync below first snapshot height" {
  run ./build/ksync height-sync -b $HOME/bins/kyved-v1.0.0 -c kaon-1 -t 50 -r -d -y
  [ "$status" -eq 0 ]
}

@test "KYVE: try to height-sync if the app has not been resetted" {
  run ./build/ksync height-sync -b $HOME/bins/kyved-v1.0.0 -c kaon-1 -t 12178 -d -y
  [ "$status" -eq 1 ]
}

@test "dYdX: height sync to specific height" {
  run ./build/ksync height-sync -b $HOME/bins/dydxprotocold-v2.0.1 -c kaon-1 -t 5935178 -r -d -y
  [ "$status" -eq 0 ]
}

@test "Archway: height sync below first snapshot height" {
  run ./build/ksync height-sync -b $HOME/bins/archwayd-v1.0.1 -t 50 -r -d -y
  [ "$status" -eq 0 ]
}

@test "Celestia: height sync to specific height" {
  run ./build/ksync height-sync -b $HOME/bins/celestia-appd-v1.3.0 -t 10010 -r -d -y
  [ "$status" -eq 0 ]
}

@test "Andromeda: height-sync to specific height" {
  run ./build/ksync height-sync -b $HOME/bins/andromedad-1-v0.1.1-beta-patch -c kaon-1 -t 2700020 -r -d -y
  [ "$status" -eq 0 ]
}
