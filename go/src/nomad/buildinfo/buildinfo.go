// Package buildinfo exposes compile-time data about the build.
package buildinfo

var (
	// These variables are initialized via the linker -X flag.
	buildSHA  string // git commit hash
	buildDate string // build time in ISO-8601 format
)

type Info struct {
	SHA  string
	Date string
}

func Get() Info {
	return Info{SHA: buildSHA, Date: buildDate}
}
