package convert

type manifestEntry struct {
	Config   string   `json:"Config"`   // path to config.json
	RepoTags []string `json:"RepoTags"` // repo:tag
	Layers   []string `json:"Layers"`   // paths to layer.tar
}
