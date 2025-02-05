package peers

import (
	"fmt"
	"github.com/KYVENetwork/ksync/types"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"os"
	"strings"
)

var (
	helpStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Margin(1, 0)
	dotStyle          = helpStyle.UnsetMargins()
	selectedItemStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("170"))
	checkMark         = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).SetString("✓")
	selectedPeers     = make([]types.Peer, 0)
)

func SelectPeers(name string, peers []types.Peer) ([]types.Peer, error) {
	_, err := tea.NewProgram(newModel(name, peers)).Run()
	if err != nil {
		return nil, err
	}

	return selectedPeers, nil
}

func SavePeers(chainSchema *types.ChainSchema, seedsArr, persistentPeersArr []types.Peer) error {
	seeds := ""
	for index, peer := range seedsArr {
		if index > 0 {
			seeds += ","
		}

		seeds += fmt.Sprintf("%s@%s", peer.Id, peer.Address)
	}

	persistentPeers := ""
	for index, peer := range persistentPeersArr {
		if index > 0 {
			persistentPeers += ","
		}

		persistentPeers += fmt.Sprintf("%s@%s", peer.Id, peer.Address)
	}

	homePath := strings.ReplaceAll(chainSchema.NodeHome, "$HOME", os.Getenv("HOME"))
	data, err := os.ReadFile(fmt.Sprintf("%s/config/config.toml", homePath))
	if err != nil {
		return err
	}
	config := make([]string, 0)

	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "seeds = ") {
			config = append(config, fmt.Sprintf("seeds = \"%s\"", seeds))
		} else if strings.HasPrefix(line, "persistent_peers = ") {
			config = append(config, fmt.Sprintf("persistent_peers = \"%s\"", persistentPeers))
		} else if strings.HasPrefix(line, "pyroscope_profile_types = ") {
			config = append(config, "pyroscope_profile_types = \"\"")
		} else {
			config = append(config, line)
		}
	}

	if err := os.WriteFile(fmt.Sprintf("%s/config/config.toml", homePath), []byte(strings.Join(config, "\n")), 0644); err != nil {
		return err
	}

	return nil
}

type model struct {
	name     string
	peers    []types.Peer
	cursor   int
	selected map[int]struct{}
	quitting bool
}

func newModel(name string, peers []types.Peer) model {
	selected := make(map[int]struct{})
	for i := range peers {
		selected[i] = struct{}{}
	}
	return model{
		name:     name,
		peers:    peers,
		selected: selected,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {

		case "enter":
			m.quitting = true
			return m, tea.Quit

		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}

		case "j", "down":
			if m.cursor < len(m.peers)-1 {
				m.cursor++
			}

		case " ":
			_, ok := m.selected[m.cursor]
			if ok {
				delete(m.selected, m.cursor)
			} else {
				m.selected[m.cursor] = struct{}{}
			}
		}
	}

	selectedPeers = make([]types.Peer, 0)

	for i := range m.selected {
		selectedPeers = append(selectedPeers, m.peers[i])
	}

	return m, nil
}

func (m model) View() string {
	if m.quitting {
		return fmt.Sprintf("%s Saved %d out of %d %s in config.toml\n", checkMark, len(m.selected), len(m.peers), m.name)
	}

	s := fmt.Sprintf("Select or deselect %s that should be included\n\n", m.name)

	for i, peer := range m.peers {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}

		checked := " "
		if _, ok := m.selected[i]; ok {
			checked = "x"
		}

		line := fmt.Sprintf("%s [%s] %s %s", cursor, checked, peer.Provider, dotStyle.Render(fmt.Sprintf("%s@%s", peer.Id, peer.Address)))
		if m.cursor == i {
			s += selectedItemStyle.Render(line) + "\n"
		} else {
			s += line + "\n"
		}
	}

	s += dotStyle.Render("\nPress space to select and enter to continue\n")
	return s
}
