package api

import (
	"context"
	"net/url"
)

// Contributor types (server: ApiV2_Contributors_List::getContributorType).
const (
	ContributorAdmin      = "administrator"
	ContributorTranslator = "contributor"
)

// Contributor is one person with access to projects: contributors/list
// (server: ApiV2_Contributors_List).
type Contributor struct {
	Name        string       `json:"name"`
	Email       string       `json:"email"`
	Permissions []Permission `json:"permissions"`
}

// Permission is a contributor's access to one project.
type Permission struct {
	Project struct {
		ID   ID     `json:"id"` // a string in this endpoint
		Name string `json:"name"`
	} `json:"project"`
	Type string `json:"type"` // ContributorAdmin or ContributorTranslator
	// Proofreader is false whenever the project has proofreading off;
	// administrators are proofreaders when it is on.
	Proofreader bool     `json:"proofreader"`
	Languages   []string `json:"languages"` // a contributor's language codes; none for an administrator
}

// ContributorsList calls contributors/list: the contributors of one project
// (projectID), or of every project of the account (projectID 0); language,
// which needs a project, keeps those of one language.
func (c *Client) ContributorsList(ctx context.Context, projectID int, language string) ([]Contributor, error) {
	params := url.Values{}
	if projectID != 0 {
		params.Set("id", idParam(projectID))
	}
	if language != "" {
		params.Set("language", language)
	}
	var result struct {
		Contributors []Contributor `json:"contributors"`
	}
	if err := c.post(ctx, "contributors/list", params, &result); err != nil {
		return nil, err
	}
	return result.Contributors, nil
}

// NewContributor is a person to add to a project (contributors/add; server:
// ApiV2_Contributors_Add, ContributorAssigner::addToProjectViaApi): an
// administrator, or a contributor to one language, maybe a proofreader.
type NewContributor struct {
	Name        string
	Email       string
	Admin       bool   // administrator of the project; Language is then ignored
	Language    string // the language a contributor translates
	Proofreader bool   // with Language; the project must have proofreading on
}

// ContributorsAdd calls contributors/add. A person already a contributor of
// that language made a proofreader is a success too.
func (c *Client) ContributorsAdd(ctx context.Context, projectID int, n NewContributor) error {
	params := url.Values{"id": {idParam(projectID)}, "name": {n.Name}, "email": {n.Email}}
	if n.Admin {
		params.Set("admin", "1")
	} else {
		params.Set("language", n.Language)
		if n.Proofreader {
			params.Set("proofreader", "1")
		}
	}
	return c.post(ctx, "contributors/add", params, nil)
}

// ContributorRemoval says what contributors/remove takes away
// (server: ApiV2_Contributors_Remove::removeUser): with nothing set, every
// access to the project (an administrator only by the owner).
type ContributorRemoval struct {
	Language    string // only this language
	Proofreader bool   // only the proofreader right
}

// The 4040 messages of contributors/remove that mean there was nothing to
// remove (server: ApiV2_Contributors_Remove::removeUser,
// Project::deleteProofreader). The language one also answers a language the
// project lacks, or a wrong code: the server overwrites 4043/4044 with it.
const (
	MsgNotInProject   = "The contributor is not assigned to this project."
	MsgNotProofreader = "This user is not a proofreader in this project."
	MsgNotInLanguage  = "The contributor is not assigned to this language."
	CodeCustomMessage = "4040"
)

// ContributorsRemove calls contributors/remove.
func (c *Client) ContributorsRemove(ctx context.Context, projectID int, email string, r ContributorRemoval) error {
	params := url.Values{"id": {idParam(projectID)}, "email": {email}}
	switch {
	case r.Proofreader:
		params.Set("proofreader", "1")
	case r.Language != "":
		params.Set("language", r.Language)
	}
	return c.post(ctx, "contributors/remove", params, nil)
}
