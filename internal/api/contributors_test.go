package api_test

import (
	"context"
	"net/url"
	"reflect"
	"testing"

	"github.com/POEditor/poeditor-cli/internal/api"
)

func TestContributorsList(t *testing.T) {
	tests := []struct {
		name     string
		project  int
		language string
		want     url.Values
	}{
		{"every project", 0, "", url.Values{}},
		{"one project", 7717, "", url.Values{"id": {"7717"}}},
		{"one language", 7717, "fr", url.Values{"id": {"7717"}, "language": {"fr"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var rec recorder
			client := rec.client(t, `{"contributors":[`+
				`{"name":"Ana","email":"ana@example.com","permissions":[{"project":{"id":"7717","name":"Web"},"type":"contributor","proofreader":true,"languages":["fr","de"]}]},`+
				`{"name":"Bo","email":"bo@example.com","permissions":[{"project":{"id":7717,"name":"Web"},"type":"administrator","proofreader":false}]}]}`)

			list, err := client.ContributorsList(context.Background(), tt.project, tt.language)

			if err != nil {
				t.Fatal(err)
			}
			rec.assertForm(t, "contributors/list", tt.want)
			if len(list) != 2 {
				t.Fatalf("list = %+v", list)
			}
			ana, bo := list[0].Permissions[0], list[1].Permissions[0]
			if list[0].Email != "ana@example.com" || ana.Project.ID != 7717 || ana.Project.Name != "Web" || ana.Type != api.ContributorTranslator ||
				!ana.Proofreader || !reflect.DeepEqual(ana.Languages, []string{"fr", "de"}) {
				t.Errorf("ana = %+v", list[0])
			}
			if bo.Project.ID != 7717 || bo.Type != api.ContributorAdmin || bo.Proofreader || bo.Languages != nil {
				t.Errorf("bo = %+v", list[1])
			}
		})
	}
}

func TestContributorsAdd(t *testing.T) {
	tests := []struct {
		name string
		n    api.NewContributor
		want url.Values
	}{
		{"contributor", api.NewContributor{Name: "Ana", Email: "ana@example.com", Language: "fr"},
			url.Values{"language": {"fr"}}},
		{"proofreader", api.NewContributor{Name: "Ana", Email: "ana@example.com", Language: "fr", Proofreader: true},
			url.Values{"language": {"fr"}, "proofreader": {"1"}}},
		{"administrator", api.NewContributor{Name: "Ana", Email: "ana@example.com", Admin: true, Language: "fr", Proofreader: true},
			url.Values{"admin": {"1"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var rec recorder
			client := rec.client(t, `null`)

			if err := client.ContributorsAdd(context.Background(), 7717, tt.n); err != nil {
				t.Fatal(err)
			}
			want := url.Values{"id": {"7717"}, "name": {"Ana"}, "email": {"ana@example.com"}}
			for k, v := range tt.want {
				want[k] = v
			}
			rec.assertForm(t, "contributors/add", want)
		})
	}
}

func TestContributorsRemove(t *testing.T) {
	tests := []struct {
		name string
		r    api.ContributorRemoval
		want url.Values
	}{
		{"everything", api.ContributorRemoval{}, url.Values{}},
		{"one language", api.ContributorRemoval{Language: "fr"}, url.Values{"language": {"fr"}}},
		{"the proofreader right", api.ContributorRemoval{Proofreader: true}, url.Values{"proofreader": {"1"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var rec recorder
			client := rec.client(t, `null`)

			if err := client.ContributorsRemove(context.Background(), 7717, "ana@example.com", tt.r); err != nil {
				t.Fatal(err)
			}
			want := url.Values{"id": {"7717"}, "email": {"ana@example.com"}}
			for k, v := range tt.want {
				want[k] = v
			}
			rec.assertForm(t, "contributors/remove", want)
		})
	}
}
