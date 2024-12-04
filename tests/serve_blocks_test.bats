@test "KYVE: serve-blocks 10 blocks from genesis" {
  run ./build/ksync serve-blocks --opt-out -b $HOME/bins/kyved-v1.0.0 --block-rpc https://rpc.kyve.network -t 10 -r -d -y
  [ "$status" -eq 0 ]
}

@test "KYVE: serve-blocks from height 10" {
  run ./build/ksync serve-blocks --opt-out -b $HOME/bins/kyved-v1.0.0 --block-rpc https://rpc.kyve.network -t 20 -d -y
  [ "$status" -eq 0 ]
}

@test "KYVE: try to serve-blocks with target height lower than current one" {
  run ./build/ksync serve-blocks --opt-out -b $HOME/bins/kyved-v1.0.0 --block-rpc https://rpc.kyve.network -t 5 -d -y
  [ "$status" -eq 1 ]
}
