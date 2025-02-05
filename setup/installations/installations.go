package installations

import (
	"compress/gzip"
	"errors"
	"fmt"
	"github.com/KYVENetwork/ksync/flags"
	"github.com/KYVENetwork/ksync/types"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

var (
	program      *tea.Program
	spinnerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
	helpStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Margin(1, 0)
	dotStyle     = helpStyle.UnsetMargins()
	checkMark    = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).SetString("✓")
	errorMark    = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).SetString("x")
	dockerLogs   string
)

func InstallGenesisSyncBinaries(chainSchema *types.ChainSchema, upgrades []types.Upgrade) error {
	program = tea.NewProgram(newModel(append([]types.Upgrade{{Name: "Cosmovisor"}}, upgrades...)))

	go func() {
		program.Run()
	}()

	homePath := strings.ReplaceAll(chainSchema.NodeHome, "$HOME", os.Getenv("HOME"))
	genesisPath := fmt.Sprintf("%s/cosmovisor/genesis/bin", homePath)

	if err := buildCosmovisor(fmt.Sprintf("%s/go/bin/", os.Getenv("HOME"))); err != nil {
		return err
	}

	if err := buildUpgradeBinary(upgrades[0], chainSchema.Codebase.GitRepoUrl, chainSchema.DaemonName, genesisPath); err != nil {
		return err
	}

	if _, err := os.Stat(fmt.Sprintf("%s/config/genesis.json", homePath)); errors.Is(err, os.ErrNotExist) {
		moniker := flags.Moniker
		if moniker == "" {
			moniker = "ksync"
		}

		cmd := exec.Command(fmt.Sprintf("%s/%s", genesisPath, chainSchema.DaemonName), "init", flags.Moniker, "--chain-id", chainSchema.ChainId)
		cmd.Env = append(os.Environ(), fmt.Sprintf("LD_LIBRARY_PATH=%s", genesisPath))

		if err := cmd.Run(); err != nil {
			program.Quit()
			program.Wait()
			return fmt.Errorf("failed to run chain init: %w", err)
		}

		out, err := os.Create(fmt.Sprintf("%s/config/genesis.json", homePath))
		if err != nil {
			program.Quit()
			program.Wait()
			return err
		}
		defer out.Close()

		resp, err := http.Get(chainSchema.Codebase.Genesis.GenesisUrl)
		if err != nil {
			program.Quit()
			program.Wait()
			return err
		}
		defer resp.Body.Close()

		data := resp.Body

		if strings.HasSuffix(chainSchema.Codebase.Genesis.GenesisUrl, ".gz") {
			data, err = gzip.NewReader(resp.Body)
			if err != nil {
				program.Quit()
				program.Wait()
				return err
			}
		}

		if _, err := io.Copy(out, data); err != nil {
			program.Quit()
			program.Wait()
			return err
		}
	}

	if _, err := os.Stat(fmt.Sprintf("%s/cosmovisor/current", homePath)); errors.Is(err, os.ErrNotExist) {
		if err := os.Symlink(fmt.Sprintf("%s/cosmovisor/genesis", homePath), fmt.Sprintf("%s/cosmovisor/current", homePath)); err != nil {
			program.Quit()
			program.Wait()
			return err
		}
	}

	for _, upgrade := range upgrades[1:] {
		outputPath := fmt.Sprintf("%s/cosmovisor/upgrades/%s/bin", homePath, upgrade.Name)

		if err := buildUpgradeBinary(upgrade, chainSchema.Codebase.GitRepoUrl, chainSchema.DaemonName, outputPath); err != nil {
			return err
		}
	}

	program.Wait()
	return nil
}

func InstallStateSyncBinaries(chainSchema *types.ChainSchema, upgrades []types.Upgrade) error {
	upgrade := upgrades[len(upgrades)-1]

	program = tea.NewProgram(newModel(append([]types.Upgrade{{Name: "Cosmovisor"}}, upgrade)))

	go func() {
		program.Run()
	}()

	homePath := strings.ReplaceAll(chainSchema.NodeHome, "$HOME", os.Getenv("HOME"))
	binaryPath := fmt.Sprintf("%s/cosmovisor/upgrades/%s/bin", homePath, upgrade.Name)

	if err := buildCosmovisor(fmt.Sprintf("%s/go/bin/", os.Getenv("HOME"))); err != nil {
		return err
	}

	if err := buildUpgradeBinary(upgrade, chainSchema.Codebase.GitRepoUrl, chainSchema.DaemonName, binaryPath); err != nil {
		return err
	}

	if _, err := os.Stat(fmt.Sprintf("%s/config/genesis.json", homePath)); errors.Is(err, os.ErrNotExist) {
		moniker := flags.Moniker
		if moniker == "" {
			moniker = "ksync"
		}

		cmd := exec.Command(fmt.Sprintf("%s/%s", binaryPath, chainSchema.DaemonName), "init", flags.Moniker, "--chain-id", chainSchema.ChainId)
		cmd.Env = append(os.Environ(), fmt.Sprintf("LD_LIBRARY_PATH=%s", binaryPath))

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to run chain init: %w", err)
		}

		out, err := os.Create(fmt.Sprintf("%s/config/genesis.json", homePath))
		if err != nil {
			return err
		}
		defer out.Close()

		resp, err := http.Get(chainSchema.Codebase.Genesis.GenesisUrl)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		data := resp.Body

		if strings.HasSuffix(chainSchema.Codebase.Genesis.GenesisUrl, ".gz") {
			data, err = gzip.NewReader(resp.Body)
			if err != nil {
				return err
			}
		}

		if _, err := io.Copy(out, data); err != nil {
			return err
		}
	}

	if _, err := os.Stat(fmt.Sprintf("%s/cosmovisor/current", homePath)); errors.Is(err, os.ErrNotExist) {
		if err := os.Symlink(fmt.Sprintf("%s/cosmovisor/upgrades/%s", homePath, upgrade.Name), fmt.Sprintf("%s/cosmovisor/current", homePath)); err != nil {
			return err
		}
	}

	program.Wait()
	return nil
}

func buildCosmovisor(outputPath string) error {
	cmd := exec.Command("docker", "build")

	if runtime.GOOS == "darwin" {
		cmd.Args = append(cmd.Args, "--platform", "linux/amd64")
		cmd.Args = append(cmd.Args, "--build-arg", fmt.Sprintf("TARGET_GOOS=%s", runtime.GOOS))
		cmd.Args = append(cmd.Args, "--build-arg", fmt.Sprintf("TARGET_GOARCH=%s", runtime.GOARCH))
	} else {
		cmd.Args = append(cmd.Args, "--platform", fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH))
		//cmd.Args = append(cmd.Args, "--build-arg", fmt.Sprintf("TARGET_GOOS=%s", ""))
		//cmd.Args = append(cmd.Args, "--build-arg", fmt.Sprintf("TARGET_GOARCH=%s", ""))
	}

	cmd.Args = append(cmd.Args, "--build-arg", "BASE_IMAGE=golang:1.23")
	cmd.Args = append(cmd.Args, "--build-arg", "VERSION=cosmovisor/v1.7.0")
	cmd.Args = append(cmd.Args, "--build-arg", "GIT_REPO=https://github.com/cosmos/cosmos-sdk")
	cmd.Args = append(cmd.Args, "--build-arg", "BINARY_PATH=cosmovisor")
	cmd.Args = append(cmd.Args, "--build-arg", "GO_VERSION=1.23")
	cmd.Args = append(cmd.Args, "--build-arg", "SUBFOLDER=tools/cosmovisor")
	cmd.Args = append(cmd.Args, "--build-arg", "BUILD_CMD=cosmovisor")

	cmd.Args = append(cmd.Args, "--output", outputPath)
	cmd.Args = append(cmd.Args, "-f", "setup/Dockerfile", ".")

	var writer CmdWriter

	cmd.Stdout = &writer
	cmd.Stderr = &writer

	start := time.Now()

	if err := cmd.Run(); err != nil {
		program.Send(fmt.Errorf(err.Error()))
		program.Quit()
		program.Wait()
		fmt.Printf("\n%s", dockerLogs)
		return fmt.Errorf("failed to run docker build: %w", err)
	}

	program.Send(types.Upgrade{
		Name:            "Cosmovisor",
		InstallDuration: time.Since(start),
	})

	return nil
}

func buildUpgradeBinary(upgrade types.Upgrade, gitRepoUrl, daemonName, outputPath string) error {
	libwasmPath := ""

	if upgrade.LibwasmVersion != "" {
		libwasmPath = fmt.Sprintf("/go/pkg/mod/github.com/!cosm!wasm/wasmvm@%s/internal/api/libwasmvm.x86_64.so", upgrade.LibwasmVersion)

		// before wasmvm v1.1.0 there was no "internal" folder yet
		libwasmVersions := strings.Split(upgrade.LibwasmVersion, ".")
		if libwasmVersions[0] == "v1" && libwasmVersions[1] == "0" {
			libwasmPath = fmt.Sprintf("/go/pkg/mod/github.com/!cosm!wasm/wasmvm@%s/api/libwasmvm.x86_64.so", upgrade.LibwasmVersion)
		}
	}

	cmd := exec.Command("docker", "build")

	if runtime.GOOS == "darwin" {
		cmd.Args = append(cmd.Args, "--platform", "linux/amd64")
		cmd.Args = append(cmd.Args, "--build-arg", fmt.Sprintf("TARGET_GOOS=%s", runtime.GOOS))
		cmd.Args = append(cmd.Args, "--build-arg", fmt.Sprintf("TARGET_GOARCH=%s", runtime.GOARCH))
	} else {
		cmd.Args = append(cmd.Args, "--platform", fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH))
		//cmd.Args = append(cmd.Args, "--build-arg", fmt.Sprintf("TARGET_GOOS=%s", ""))
		//cmd.Args = append(cmd.Args, "--build-arg", fmt.Sprintf("TARGET_GOARCH=%s", ""))
	}

	cmd.Args = append(cmd.Args, "--build-arg", fmt.Sprintf("BASE_IMAGE=golang:%s", upgrade.GoVersion))
	cmd.Args = append(cmd.Args, "--build-arg", fmt.Sprintf("VERSION=%s", upgrade.Version))
	cmd.Args = append(cmd.Args, "--build-arg", fmt.Sprintf("GIT_REPO=%s", gitRepoUrl))
	cmd.Args = append(cmd.Args, "--build-arg", fmt.Sprintf("BINARY_PATH=%s", fmt.Sprintf("build/%s", daemonName)))
	cmd.Args = append(cmd.Args, "--build-arg", fmt.Sprintf("GO_VERSION=%s", upgrade.GoVersion))

	if libwasmPath != "" {
		cmd.Args = append(cmd.Args, "--build-arg", fmt.Sprintf("LIBWASM_PATH=%s", libwasmPath))
	}

	cmd.Args = append(cmd.Args, "--output", outputPath)
	cmd.Args = append(cmd.Args, "-f", "setup/Dockerfile", ".")

	var writer CmdWriter

	cmd.Stdout = &writer
	cmd.Stderr = &writer

	start := time.Now()

	if err := cmd.Run(); err != nil {
		program.Send(fmt.Errorf(err.Error()))
		program.Quit()
		program.Wait()
		fmt.Printf("\n%s", dockerLogs)
		return fmt.Errorf("failed to run docker build: %w", err)
	}

	upgrade.InstallDuration = time.Since(start)
	program.Send(upgrade)

	return nil
}

type CmdWriter struct{}

func (w *CmdWriter) Write(p []byte) (n int, err error) {
	dockerLogs += string(p) + "\n"
	return len(p), nil
}

type model struct {
	spinner           spinner.Model
	currentUpgrade    string
	logs              []string
	upgrades          []types.Upgrade
	installedUpgrades []types.Upgrade
	error             bool
}

func newModel(upgrades []types.Upgrade) model {
	const numLastResults = 10
	s := spinner.New()
	s.Style = spinnerStyle
	s.Spinner = spinner.Dot
	return model{
		spinner:           s,
		logs:              make([]string, numLastResults),
		upgrades:          upgrades,
		installedUpgrades: make([]types.Upgrade, 0),
		error:             false,
	}
}

func (m model) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

		return m, nil
	case types.Upgrade:
		m.installedUpgrades = append(m.installedUpgrades, msg)
		if len(m.installedUpgrades) == len(m.upgrades) {
			return m, tea.Quit
		}
		return m, nil
	case error:
		m.error = true
		return m, tea.Quit
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	default:
		return m, nil
	}
}

func (m model) View() string {
	var s string

	lastIndex := -1

	for index, upgrade := range m.installedUpgrades {
		if upgrade.Name == "Cosmovisor" {
			s += fmt.Sprintf("%s Installed %s %s\n", checkMark, upgrade.Name, dotStyle.Render(upgrade.InstallDuration.String()))
		} else {
			s += fmt.Sprintf("%s Installed upgrade %s %s\n", checkMark, upgrade.Name, dotStyle.Render(upgrade.InstallDuration.String()))
		}
		lastIndex = index
	}

	for index, upgrade := range m.upgrades {
		if lastIndex >= index {
			continue
		}

		if upgrade.Name == "Cosmovisor" {
			if lastIndex+1 == index {
				if m.error {
					s += fmt.Sprintf("%s Failed to install %s\n", errorMark, upgrade.Name)
				} else {
					s += m.spinner.View() + fmt.Sprintf("Installing %s ...\n", upgrade.Name)
				}
			} else {
				s += fmt.Sprintf("  Scheduled %s\n", upgrade.Name)
			}
		} else {
			if lastIndex+1 == index {
				if m.error {
					s += fmt.Sprintf("%s Failed to install %s\n", errorMark, upgrade.Name)
				} else {
					s += m.spinner.View() + fmt.Sprintf("Installing upgrade %s ...\n", upgrade.Name)
				}
			} else {
				s += fmt.Sprintf("  Scheduled upgrade %s\n", upgrade.Name)
			}
		}
	}

	//if len(m.installedUpgrades) < len(m.upgrades) {
	//	s += "\n"
	//
	//	for _, log := range m.logs {
	//		s += log + "\n"
	//	}
	//}

	return s
}
