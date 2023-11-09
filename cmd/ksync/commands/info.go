package commands

import (
	_ "embed"
	"fmt"
	"github.com/KYVENetwork/ksync/sources"
	"github.com/KYVENetwork/ksync/utils"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
	"os"
	"sort"
)

var registryUrl string

func init() {
	infoCmd.Flags().StringVar(&chainId, "chain-id", utils.DefaultChainId, fmt.Sprintf("KYVE chain id [\"%s\",\"%s\"]", utils.ChainIdMainnet, utils.ChainIdKaon))

	infoCmd.Flags().StringVar(&registryUrl, "registry-url", utils.DefaultRegistryURL, "URL to fetch latest KYVE Source-Registry")

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

		sourceRegistry, err := sources.GetSourceRegistry(registryUrl)
		if err != nil {
			logger.Error().Str("err", err.Error()).Msg("failed to get source registry")
			return
		}

		// Sort SourceRegistry
		sortFunc := func(keys []string) {
			sort.Slice(keys, func(i, j int) bool {
				return sourceRegistry.Entries[keys[i]].Source.ChainID < sourceRegistry.Entries[keys[j]].Source.ChainID
			})
		}
		keys := make([]string, 0, len(sourceRegistry.Entries))
		for key := range sourceRegistry.Entries {
			keys = append(keys, key)
		}
		sortFunc(keys)

		t := table.NewWriter()
		t.SetOutputMirror(os.Stdout)
		t.AppendHeader(table.Row{"Source", "BLOCK-SYNC", "STATE-SYNC", "HEIGHT-SYNC"})

		for _, key := range keys {
			entry := sourceRegistry.Entries[key]

			if chainId == utils.ChainIdMainnet {
				if (entry.Kyve.StatePoolID == nil) && (entry.Kyve.BlockPoolID == nil) {
					continue
				}
			} else if chainId == utils.ChainIdKaon {
				if (entry.Kaon.StatePoolID == nil) && (entry.Kaon.BlockPoolID == nil) {
					continue
				}
			}
			blockSync, stateSync, heightSync := sources.FormatOutput(&entry, chainId)
			t.AppendRows([]table.Row{
				{
					entry.Source.Title,
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
