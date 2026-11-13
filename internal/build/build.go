// Package build carries values stamped in at link time by the release build:
//
//	go build -ldflags "-X github.com/POEditor/poeditor-cli/internal/build.Version=1.2.3"
package build

// Version is the CLI version; "dev" for local builds.
var Version = "dev"
