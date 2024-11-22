package version

type ok struct {
	Version   string `json:"version" example:"v0.1.0"`
	Commit    string `json:"commit" example:"<commit-hash>"`
	BuildTime string `json:"buildTime" example:"2023-01-01T00:00:00Z"`
	Dirty     bool   `json:"dirty"`
}
