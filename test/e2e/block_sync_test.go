package e2e_test

import (
	"fmt"
	cmd "github.com/KYVENetwork/ksync/cmd/ksync/commands"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"os"
	"testing"
)

func TestBlockSync(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "block-sync")
}

var _ = Describe("block-sync", Ordered, func() {
	var homePath string

	BeforeAll(func() {
		h, err := os.UserHomeDir()
		Expect(err).ShouldNot(HaveOccurred())

		homePath = h
		cmd.RootCmd.PersistentFlags().BoolP("help", "", false, "help for this command")
	})

	It("KYVE: block sync 50 blocks from genesis", func() {
		// cmd.RootCmd.SetArgs([]string{"block-sync", "--binary", fmt.Sprintf("%s/bins/kyved-v1.0.0", homePath), "--chain-id", "kaon-1", "-t", "50", "-y"})
		os.Args = []string{os.Args[0], "block-sync", "--binary", fmt.Sprintf("%s/bins/kyved-v1.0.0", homePath), "--chain-id", "kaon-1", "-t", "50", "-y"}

		err := cmd.RootCmd.Execute()
		Expect(err).ShouldNot(HaveOccurred())
	})

	It("KYVE: info command", func() {
		os.Args = []string{os.Args[0], "info"}

		err := cmd.RootCmd.Execute()
		Expect(err).ShouldNot(HaveOccurred())
	})

	It("KYVE: continue block sync from height 50", func() {
		//cmd.RootCmd.SetArgs([]string{"block-sync", "--binary", fmt.Sprintf("%s/bins/kyved-v1.0.0", homePath), "--chain-id", "kaon-1", "-t", "100", "-y"})
		os.Args = []string{os.Args[0], "block-sync", "--binary", fmt.Sprintf("%s/bins/kyved-v1.0.0", homePath), "--chain-id", "kaon-1", "-t", "100", "-y"}

		err := cmd.RootCmd.Execute()
		Expect(err).ShouldNot(HaveOccurred())
	})
})
