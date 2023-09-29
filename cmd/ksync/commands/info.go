package commands

import (
	_ "embed"
	"fmt"
	"github.com/KYVENetwork/ksync/chains"
	"github.com/KYVENetwork/ksync/utils"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
	"os"
)

func init() {
	infoCmd.Flags().StringVar(&chainId, "chain-id", utils.DefaultChainId, fmt.Sprintf("KYVE chain id [\"%s\",\"%s\"]", utils.ChainIdMainnet, utils.ChainIdKaon))

	rootCmd.AddCommand(infoCmd)
}

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Get KSYNC chain support information",
	Run: func(cmd *cobra.Command, args []string) {
		if chainId != utils.ChainIdMainnet && chainId != utils.ChainIdKaon {
			logger.Error().Str("chain-id", chainId).Msg("chain information not supported")
			return
		}

		supportedChains, err := chains.GetSupportedChains(chainId)
		if err != nil {
			logger.Error().Str("err", err.Error()).Msg("failed to get supported chains")
			return
		}

		t := table.NewWriter()
		t.SetOutputMirror(os.Stdout)
		t.AppendHeader(table.Row{"Chain", "Block height [ID]", "State height [ID]", "BLOCK-SYNC", "STATE-SYNC", "HEIGHT-SYNC"})

		for _, c := range *supportedChains {
			blockKey, stateKey, blockSync, stateSync, heightSync := chains.FormatOutput(&c)
			t.AppendRows([]table.Row{
				{
					c.Name,
					blockKey,
					stateKey,
					blockSync,
					stateSync,
					heightSync,
				},
			})
		}

		t.SetStyle(table.StyleRounded)
		t.Render()
	},
}
