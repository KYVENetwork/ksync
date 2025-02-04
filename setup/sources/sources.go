package sources

import (
	"encoding/json"
	"fmt"
	"github.com/KYVENetwork/ksync/flags"
	"github.com/KYVENetwork/ksync/utils"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"io"
	"os"
	"strings"
)

const defaultWidth = 20
const listHeight = 14

var (
	titleStyle        = lipgloss.NewStyle().MarginLeft(2)
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
	paginationStyle   = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle         = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
	checkMark         = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).SetString("✓")
	errorMark         = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).SetString("x")
)

type item struct {
	title string
	desc  string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return "" }
func (i item) FilterValue() string { return i.title }

type itemDelegate struct{}

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}

	str := fmt.Sprintf("%d. %s", index+1, i.title)

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("> " + strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(str))
}

func SelectSource() error {
	if flags.Source != "" {
		return nil
	}

	sourcesResponse, err := utils.GetFromUrlWithErr("https://api.github.com/repos/cosmos/chain-registry/contents")
	if err != nil {
		return err
	}

	var sources []struct {
		Name string `json:"name"`
		Type string `json:"type"`
	}

	if err := json.Unmarshal(sourcesResponse, &sources); err != nil {
		return err
	}

	options := make([]list.Item, 0)

	for _, s := range sources {
		if s.Type != "dir" {
			continue
		}

		if strings.HasPrefix(s.Name, "_") {
			continue
		}

		if strings.HasPrefix(s.Name, ".") {
			continue
		}

		options = append(options, list.Item(item{title: s.Name, desc: ""}))
	}

	if _, err := tea.NewProgram(newModel(options)).Run(); err != nil {
		return err
	}

	if flags.Source == "" {
		os.Exit(0)
	}

	return nil
}

type model struct {
	list     list.Model
	quitting bool
}

func newModel(options []list.Item) model {
	l := list.New(options, itemDelegate{}, defaultWidth, listHeight)
	l.Title = "Select chain?"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle

	return model{
		list:     list.New(options, itemDelegate{}, defaultWidth, listHeight),
		quitting: false,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case "enter":
			i, ok := m.list.SelectedItem().(item)
			if ok {
				flags.Source = i.title
			}
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if flags.Source != "" {
		return fmt.Sprintf("%s Selected chain %s\n", checkMark, flags.Source)
	}
	if m.quitting {
		return fmt.Sprintf("%s Skipped selecting chain\n", errorMark)
	}
	return "\n" + m.list.View()
}
