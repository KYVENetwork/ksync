@test "KYVE: serve-snapshots from start height to target height" {
  run ./build/ksync serve-snapshots --opt-out -b $HOME/bins/kyved-v1.0.0 -c kaon-1 --start-height 3000 -t 3050 -r -d
  [ "$status" -eq 0 ]
}

@test "KYVE: continue serve-snapshots from current height" {
  run ./build/ksync serve-snapshots --opt-out -b $HOME/bins/kyved-v1.0.0 -c kaon-1 -t 3100 -d
  [ "$status" -eq 0 ]
}
