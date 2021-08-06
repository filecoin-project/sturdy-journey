package build

var CurrentCommit string

const version = "v0.2.0-dev"

func Version() string {
	return version + CurrentCommit
}
