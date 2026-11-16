package transfer

import "github.com/POEditor/poeditor-cli/internal/repoconfig"

// Status is what happened to one file in one language (the JSON "status").
type Status string

// Pull statuses. Push adds its own (push.Uploaded…).
const (
	Created     Status = "created"
	Updated     Status = "updated"
	Unchanged   Status = "unchanged"
	WouldCreate Status = "would_create"
	WouldUpdate Status = "would_update"
)

// dryRun is the status a dry run reports instead of s.
func (s Status) dryRun() Status {
	switch s {
	case Created:
		return WouldCreate
	case Updated:
		return WouldUpdate
	}
	return s
}

// FileResult is one file in one language of a pull or push, as reported by
// --json. Push embeds it and adds the upload counts.
type FileResult struct {
	Project  repoconfig.ProjectRef `json:"project"`
	File     string                `json:"file"`
	Language string                `json:"language"`
	Status   Status                `json:"status"`
}

// Result returns the item's FileResult with status.
func (it Item) Result(status Status) FileResult {
	return FileResult{Project: it.Project.Ref(), File: it.Display, Language: it.Language, Status: status}
}
