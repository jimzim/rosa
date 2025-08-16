package version

// Build information set at compile time
var (
	// Version is the semantic version
	Version = "2.0.0-dev"

	// BuildTime is the build timestamp
	BuildTime = "unknown"

	// GitCommit is the git commit hash
	GitCommit = "unknown"

	// GoVersion is the Go version used to build
	GoVersion = "unknown"
)

// Info returns version information as a map
func Info() map[string]string {
	return map[string]string{
		"Version":   Version,
		"BuildTime": BuildTime,
		"GitCommit": GitCommit,
		"GoVersion": GoVersion,
		"Edition":   "HCP-Only",
	}
}

// String returns a formatted version string
func String() string {
	return Version + " (HCP-Only Edition)"
}
