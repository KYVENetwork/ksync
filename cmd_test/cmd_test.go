package cmd_test

import (
	cmd "github.com/KYVENetwork/ksync/cmd/ksync/commands"
	"gotest.tools/assert"
	"os"
	"testing"
)

func Test(t *testing.T) {
	cmd.RootCmd.SetOut(os.Stdout)
	cmd.RootCmd.SetErr(os.Stdout)
	cmd.RootCmd.PersistentFlags().BoolP("help", "", false, "help for this command")
	cmd.RootCmd.SetArgs([]string{"block-sync", "--binary", "/Users/troykessler/work/kyve/bins/kyved-v1.0.0", "--chain-id", "kaon-1", "-t", "100", "-r", "-y"})

	err := cmd.RootCmd.Execute()

	assert.Equal(t, err, nil, "error is not nil")
}
