@test "KYVE: version" {
  run ./build/ksync version --opt-out
  [ "$status" -eq 0 ]
}
