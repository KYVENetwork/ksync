package cmd_test

import (
	cmd "github.com/KYVENetwork/ksync/cmd/ksync/commands"
	"testing"

	"github.com/matryer/is"
)

func Test(t *testing.T) {
	code := cmd.RunVersionCmd([]string{})

	is.New(t).True(code == 0)
}
