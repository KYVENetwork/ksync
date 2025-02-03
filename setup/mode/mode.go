package mode

import (
	"fmt"
	"github.com/KYVENetwork/ksync/types"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	spinnerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
	helpStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Margin(1, 0)
	dotStyle     = helpStyle.UnsetMargins()
	checkMark    = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).SetString("✓")
	setupMode    int
)

func SelectSetupMode() (*types.ChainSchema, []types.Upgrade, int, error) {
	p := tea.NewProgram(newModel())

	go func() {
		p.Run()
	}()

	chainSchema, err := FetchChainSchema()
	if err != nil {
		return nil, nil, 0, err
	}

	upgrades, err := FetchUpgrades(chainSchema)
	if err != nil {
		return nil, nil, 0, err
	}

	p.Send(true)

	p.Wait()

	return chainSchema, upgrades, setupMode, nil
}

type model struct {
	spinner  spinner.Model
	cursor   int
	modes    []string
	loaded   bool
	quitting bool
}

func newModel() model {
	s := spinner.New()
	s.Style = spinnerStyle
	s.Spinner = spinner.Dot
	return model{
		spinner: s,
		modes: []string{
			"1. Install binary with Cosmovisor from source",
			"2. Install binaries and state-sync to live height",
			"3. Install binaries and block-sync from genesis to live height",
			"4. Exit",
		},
		loaded:   false,
		quitting: false,
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
			setupMode = m.cursor + 1
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
	case bool:
		m.loaded = msg
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
			return fmt.Sprintf("%s Selected binary installation with state-sync to live height\n", checkMark)
		} else if setupMode == 3 {
			return fmt.Sprintf("%s Selected binary installation with block-sync from genesis to live height\n", checkMark)
		} else {
			return fmt.Sprintf("%s Selected exit\n", checkMark)
		}
	} else if !m.loaded {
		return m.spinner.View() + " Loading chain information ..."
	}

	s := fmt.Sprintf("Select the setup mode for your chain\n\n")

	for i, mode := range m.modes {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}

		s += fmt.Sprintf("%s %s\n", cursor, mode)
	}

	s += dotStyle.Render("\nPress enter to select\n")
	return s
}
