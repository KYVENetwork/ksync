package setup

type Peer struct {
	Id       string `json:"id"`
	Address  string `json:"address"`
	Provider string `json:"provider"`
}

type Endpoint struct {
	Address  string `json:"address"`
	Provider string `json:"provider"`
}

type FeeToken struct {
	Denom       string  `json:"denom"`
	LowGasPrice float64 `json:"low_gas_price"`
}

type ChainSchema struct {
	ChainId    string `json:"chain_id"`
	DaemonName string `json:"daemon_name"`
	Fees       struct {
		FeeTokens []FeeToken `json:"fee_tokens"`
	} `json:"fees"`
	Codebase struct {
		GitRepoUrl string `json:"git_repo"`
		Genesis    struct {
			GenesisUrl string `json:"genesis_url"`
		} `json:"genesis"`
	} `json:"codebase"`
	Peers struct {
		Seeds           []Peer `json:"seeds"`
		PersistentPeers []Peer `json:"persistent_peers"`
	} `json:"peers"`
	Apis struct {
		Rpc  []Endpoint `json:"rpc"`
		Rest []Endpoint `json:"rest"`
	} `json:"apis"`
}

type Version struct {
	Name               string `json:"name"`
	RecommendedVersion string `json:"recommended_version"`
	Tag                string `json:"tag"`
}

type VersionsSchema struct {
	Versions []Version `json:"versions"`
}
