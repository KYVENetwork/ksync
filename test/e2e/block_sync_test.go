package e2e_test

import (
	"fmt"
	cmd "github.com/KYVENetwork/ksync/cmd/ksync/commands"
	"gotest.tools/assert"
	"os"
	"testing"
)

func Test(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	assert.NilError(t, err)

	cmd.RootCmd.PersistentFlags().BoolP("help", "", false, "help for this command")
	cmd.RootCmd.SetArgs([]string{"block-sync", "--binary", fmt.Sprintf("%s/bins/kyved-v1.0.0", homeDir), "--chain-id", "kaon-1", "-t", "100", "-r", "-y"})

	err = cmd.RootCmd.Execute()

	assert.Equal(t, err, nil, "error is not nil")
}
