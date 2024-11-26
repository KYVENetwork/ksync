package commands

import (
	_ "embed"
	"fmt"
	"github.com/KYVENetwork/ksync/sources"
	"github.com/KYVENetwork/ksync/sources/helpers"
	"github.com/KYVENetwork/ksync/utils"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
	"os"
	"sort"
)

func init() {
	infoCmd.Flags().StringVarP(&chainId, "chain-id", "c", utils.DefaultChainId, fmt.Sprintf("KYVE chain id [\"%s\",\"%s\"]", utils.ChainIdMainnet, utils.ChainIdKaon))

	infoCmd.Flags().StringVar(&registryUrl, "registry-url", utils.DefaultRegistryURL, "URL to fetch latest KYVE Source-Registry")

	infoCmd.Flags().BoolVar(&optOut, "opt-out", false, "disable the collection of anonymous usage data")

	RootCmd.AddCommand(infoCmd)
}

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Get KSYNC chain support information",
	RunE: func(cmd *cobra.Command, args []string) error {
		utils.TrackInfoEvent(chainId, optOut)

		if chainId != utils.ChainIdMainnet && chainId != utils.ChainIdKaon {
			return fmt.Errorf("chain-id %s not supported", chainId)
		}

		sourceRegistry, err := helpers.GetSourceRegistry(registryUrl)
		if err != nil {
			return fmt.Errorf("failed to get source registry: %w", err)
		}

		// Sort SourceRegistry
		sortFunc := func(keys []string) {
			sort.Slice(keys, func(i, j int) bool {
				return sourceRegistry.Entries[keys[i]].SourceID < sourceRegistry.Entries[keys[j]].SourceID
			})
		}

		var keys []string
		for key, entry := range sourceRegistry.Entries {
			if chainId == utils.ChainIdMainnet {
				if entry.Networks.Kyve != nil {
					if entry.Networks.Kyve.Integrations != nil {
						if entry.Networks.Kyve.Integrations.KSYNC != nil {
							keys = append(keys, key)
						}
					}
				}
			}
			if chainId == utils.ChainIdKaon {
				if entry.Networks.Kaon != nil {
					if entry.Networks.Kaon.Integrations != nil {
						if entry.Networks.Kaon.Integrations.KSYNC != nil {
							keys = append(keys, key)
						}
					}
				}
			}
		}
		sortFunc(keys)

		t := table.NewWriter()
		t.SetOutputMirror(os.Stdout)
		t.AppendHeader(table.Row{"Source", "BLOCK-SYNC", "STATE-SYNC", "HEIGHT-SYNC"})

		for _, key := range keys {
			entry := sourceRegistry.Entries[key]

			var title string

			if chainId == utils.ChainIdMainnet {
				if entry.Networks.Kyve != nil {
					if entry.Networks.Kyve.Integrations != nil {
						if entry.Networks.Kyve.Integrations.KSYNC == nil {
							continue
						}
						title = entry.Networks.Kyve.SourceMetadata.Title
					}
				} else {
					continue
				}
			} else if chainId == utils.ChainIdKaon {
				if entry.Networks.Kaon != nil {
					if entry.Networks.Kaon.Integrations != nil {
						if entry.Networks.Kaon.Integrations.KSYNC == nil {
							continue
						}
						title = entry.Networks.Kaon.SourceMetadata.Title
					}
				} else {
					continue
				}
			}

			blockSync, stateSync, heightSync := sources.FormatOutput(&entry, chainId)
			t.AppendRows([]table.Row{
				{
					title,
					blockSync,
					stateSync,
					heightSync,
				},
			})
		}

		t.SetStyle(table.StyleRounded)
		t.Render()
		return nil
	},
}
