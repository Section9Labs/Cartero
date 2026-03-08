package version

type Info struct {
	Version string
	Commit  string
	Date    string
}

var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

func BuildInfo() Info {
	return Info{
		Version: Version,
		Commit:  Commit,
		Date:    Date,
	}
}
