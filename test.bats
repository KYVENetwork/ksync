@test "KYVE: block sync 50 blocks from genesis" {
  run ./build/ksync block-sync -b $HOME/bins/kyved-v1.0.0 -c kaon-1 -t 50 -r -y
  [ "$status" -eq 0 ]
}

@test "KYVE: continue block sync from height 50" {
  run ./build/ksync block-sync -b $HOME/bins/kyved-v1.0.0 -c kaon-1 -t 100 -y
  [ "$status" -eq 0 ]
}
