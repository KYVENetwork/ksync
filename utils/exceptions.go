package utils

type exception struct {
	Subfolder  string
	BuildCmd   string
	BinaryPath string
}

var Exceptions = map[string]exception{
	"dydx-mainnet-1": {
		Subfolder: "protocol",
	},
	"noble-1": {
		BinaryPath: "build",
	},
	"andromeda-1": {
		BinaryPath: "bin/andromedad",
	},
	"source-1": {
		BinaryPath: "bin/sourced",
	},
	"axelar-dojo-1": {
		BinaryPath: "bin/axelard",
	},
	"zetachain_7000-1": {
		BuildCmd:   "install-zetacore",
		BinaryPath: "/usr/local/go/bin/zetacored",
	},
}
