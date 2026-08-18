package buildinfo

// These values are overridden at build time with -ldflags.
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
)

type Info struct {
	Product   string `json:"product"`
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildTime string `json:"build_time"`
}

func Current() Info {
	return Info{
		Product:   "HermesDDNS",
		Version:   Version,
		Commit:    Commit,
		BuildTime: BuildTime,
	}
}
