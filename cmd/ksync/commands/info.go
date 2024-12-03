package commands

import (
	_ "embed"
	"fmt"
	"github.com/KYVENetwork/ksync/app/source"
	"github.com/KYVENetwork/ksync/flags"
	"github.com/KYVENetwork/ksync/utils"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
	"os"
	"sort"
)

func init() {
	infoCmd.Flags().StringVarP(&flags.ChainId, "chain-id", "c", utils.DefaultChainId, fmt.Sprintf("KYVE chain id [\"%s\",\"%s\"]", utils.ChainIdMainnet, utils.ChainIdKaon))

	infoCmd.Flags().BoolVar(&flags.OptOut, "opt-out", false, "disable the collection of anonymous usage data")
	infoCmd.Flags().BoolVarP(&flags.Debug, "debug", "d", false, "run KSYNC in debug mode")

	RootCmd.AddCommand(infoCmd)
}

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Get KSYNC chain support information",
	RunE: func(cmd *cobra.Command, args []string) error {
		if flags.ChainId != utils.ChainIdMainnet && flags.ChainId != utils.ChainIdKaon {
			return fmt.Errorf("chain-id %s not supported", flags.ChainId)
		}

		sourceRegistry, err := source.GetSourceRegistry(utils.DefaultRegistryURL)
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
			if flags.ChainId == utils.ChainIdMainnet {
				if entry.Networks.Kyve != nil {
					if entry.Networks.Kyve.Integrations != nil {
						if entry.Networks.Kyve.Integrations.KSYNC != nil {
							keys = append(keys, key)
						}
					}
				}
			}
			if flags.ChainId == utils.ChainIdKaon {
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

			if flags.ChainId == utils.ChainIdMainnet {
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
			} else if flags.ChainId == utils.ChainIdKaon {
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

			blockSync, stateSync, heightSync := source.FormatOutput(&entry, flags.ChainId)
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
