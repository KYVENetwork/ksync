package utils

type exception struct {
	Subfolder  string
	BinaryPath string
}

var Exceptions = map[string]exception{
	"dydx-mainnet-1": {
		Subfolder:  "protocol",
		BinaryPath: "",
	},
	"noble-1": {
		Subfolder:  "",
		BinaryPath: "build",
	},
	"andromeda-1": {
		Subfolder:  "",
		BinaryPath: "bin/andromedad",
	},
	"source-1": {
		Subfolder:  "",
		BinaryPath: "bin/sourced",
	},
}
