package build

var CurrentCommit string

const version = "v0.1.0"

func Version() string {
	return version + CurrentCommit
}
