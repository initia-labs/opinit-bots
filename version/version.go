package version

var (
	// Version is the current version of the bot
	Version = ""

	// GitCommit is the commit hash of the bot
	GitCommit = ""
)

type Info struct {
	Version   string `json:"version" yaml:"version"`
	GitCommit string `json:"git_commit" yaml:"git-commit"`
}

func NewInfo() Info {
	return Info{
		Version:   Version,
		GitCommit: GitCommit,
	}
}
