@test "KYVE: info on mainnet" {
  run ./build/ksync info --opt-out
  [ "$status" -eq 0 ]
}

@test "KYVE: info on testnet" {
  run ./build/ksync info --opt-out --chain-id kaon-1
  [ "$status" -eq 0 ]
}

@test "KYVE: info on devnet" {
  run ./build/ksync info --opt-out --chain-id korellia-2
  [ "$status" -eq 1 ]
}
