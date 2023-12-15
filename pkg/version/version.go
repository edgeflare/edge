package version

import "strings"

var (
	// Program is the name of the program
	Program = "edge"
	// ProgramUpper is the name of the program in uppercase
	ProgramUpper = strings.ToUpper(Program)
	// Version is the version of the program
	Version = "v0.0.1-alpha1"
	// GitCommit is the git commit of the program
	GitCommit = "HEAD"
)
