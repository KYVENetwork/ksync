package mode

import (
	"fmt"
	"github.com/KYVENetwork/ksync/app/collector"
	"github.com/KYVENetwork/ksync/app/source"
	"github.com/KYVENetwork/ksync/flags"
	"github.com/KYVENetwork/ksync/types"
	"github.com/KYVENetwork/ksync/utils"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"runtime"
	"strings"
	"time"
)

var (
	spinnerStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
	helpStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Margin(1, 0)
	selectedItemStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("170"))
	dotStyle          = helpStyle.UnsetMargins()
	checkMark         = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).SetString("✓")
	errorMark         = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).SetString("x")
	setupMode         int
)

func SelectSetupMode() (*types.ChainSchema, []types.Upgrade, int, error) {
	p := tea.NewProgram(newModel())
	go func() {
		p.Run()
	}()

	chainSchema, err := FetchChainSchema()
	if err != nil {
		p.Quit()
		p.Wait()
		return nil, nil, 0, err
	}

	sourceInfo, err := source.NewSource(chainSchema.ChainId)
	if err != nil {
		p.Quit()
		p.Wait()
		return nil, nil, 0, err
	}

	upgrades, err := FetchUpgrades(chainSchema)
	if err != nil {
		p.Quit()
		p.Wait()
		return nil, nil, 0, err
	}

	canRunDarwin := true
	for _, upgrade := range upgrades {
		if upgrade.LibwasmVersion != "" {
			canRunDarwin = false
		}
	}

	if runtime.GOOS == "darwin" && !canRunDarwin {
		p.Quit()
		p.Wait()
		return nil, nil, 0, fmt.Errorf("chain binaries contain cosmwasm, unable to cross-compile for darwin")
	}

	modes := []string{"1. Install binary with Cosmovisor from source"}

	chainRest, err := func() (string, error) {
		if flags.ChainRest != "" {
			return strings.TrimSuffix(flags.ChainRest, "/"), nil
		}

		switch flags.ChainId {
		case utils.ChainIdMainnet:
			return utils.RestEndpointMainnet, nil
		case utils.ChainIdKaon:
			return utils.RestEndpointKaon, nil
		case utils.ChainIdKorellia:
			return utils.RestEndpointKorellia, nil
		default:
			return "", fmt.Errorf("flag --chain-id has to be either \"%s\", \"%s\" or \"%s\"", utils.ChainIdMainnet, utils.ChainIdKaon, utils.ChainIdKorellia)
		}
	}()
	if err != nil {
		p.Quit()
		p.Wait()
		return nil, nil, 0, err
	}

	if poolId, err := sourceInfo.GetSourceSnapshotPoolId(); err == nil {
		snapshotCollector, err := collector.NewKyveSnapshotCollector(poolId, chainRest)
		if err != nil {
			p.Quit()
			p.Wait()
			return nil, nil, 0, err
		}
		modes = append(modes, fmt.Sprintf("2. Install binaries and state-sync to latest height %d", snapshotCollector.GetLatestAvailableHeight()))
	}

	if _, err := sourceInfo.GetSourceBlockPoolId(); err == nil {
		modes = append(modes, "3. Install binaries and block-sync from genesis to live height")
	}

	modes = append(modes, fmt.Sprintf("%d. Exit", len(modes)+1))

	height, err := FetchLatestHeight(chainSchema)
	if err == nil {
		p.Send(height)
	}

	go func() {
		for {
			time.Sleep(5 * time.Second)

			height, err = FetchLatestHeight(chainSchema)
			if err == nil {
				p.Send(height)
			}
		}
	}()

	p.Send(modes)
	p.Wait()

	return chainSchema, upgrades, setupMode, nil
}

type model struct {
	spinner      spinner.Model
	cursor       int
	modes        []string
	quitting     bool
	latestHeight int64
}

func newModel() model {
	s := spinner.New()
	s.Style = spinnerStyle
	s.Spinner = spinner.Dot

	return model{
		spinner:      s,
		modes:        make([]string, 0),
		quitting:     false,
		latestHeight: 0,
	}
}

func (m model) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {

		case "enter":
			if strings.Contains(m.modes[m.cursor], "Exit") {
				setupMode = 0
			} else {
				setupMode = m.cursor + 1
			}

			m.quitting = true
			return m, tea.Quit

		case "up":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down":
			if m.cursor < len(m.modes)-1 {
				m.cursor++
			}
		}
	case int64:
		m.latestHeight = msg
		return m, nil
	case []string:
		m.modes = msg
		return m, nil
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	default:
		return m, nil
	}

	return m, nil
}

func (m model) View() string {
	if m.quitting {
		if setupMode == 1 {
			return fmt.Sprintf("%s Selected binary installation\n", checkMark)
		} else if setupMode == 2 {
			return fmt.Sprintf("%s Selected binary installation with state-sync to latest height\n", checkMark)
		} else if setupMode == 3 {
			return fmt.Sprintf("%s Selected binary installation with block-sync from genesis to live height\n", checkMark)
		} else {
			return fmt.Sprintf("%s Selected exit\n", errorMark)
		}
	} else if len(m.modes) == 0 {
		return m.spinner.View() + fmt.Sprintf("Loading chain information for %s ...", flags.Source)
	}

	s := fmt.Sprintf("Select the setup mode for your chain\n\n")

	for i, mode := range m.modes {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}

		var line string

		if m.latestHeight > 0 && i > 0 && i < len(m.modes)-1 {
			line = fmt.Sprintf("%s %s %s", cursor, mode, dotStyle.Render(fmt.Sprintf("(live height %d)", m.latestHeight)))
		} else {
			line = fmt.Sprintf("%s %s", cursor, mode)
		}

		if m.cursor == i {
			s += selectedItemStyle.Render(line) + "\n"
		} else {
			s += line + "\n"
		}
	}

	s += dotStyle.Render("\nPress enter to select\n")
	return s
}
