@test "KYVE: version" {
  run ./build/ksync version
  [ "$status" -eq 0 ]
}
