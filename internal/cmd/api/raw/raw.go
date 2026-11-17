// Package raw gives every API v2 endpoint that has no dedicated command yet
// a command of its own under `poeditor api` (`poeditor api contributors list
// --id 7717`): it sends the request with the account's token and prints the
// response. When an endpoint gets its dedicated command, that command takes
// its place, so each endpoint is always exactly one command.
package raw

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/POEditor/poeditor-cli/internal/api"
	"github.com/POEditor/poeditor-cli/internal/cmdutil"
)

// Endpoints is every public API v2 endpoint, <group>/<action> (server:
// application/modules/v2/controllers; listAll is list, addComment is
// add_comment). translations/propose has a model but no controller action,
// so it is not public.
var Endpoints = []string{
	"projects/list", "projects/add", "projects/view", "projects/update", "projects/delete",
	"projects/upload", "projects/sync", "projects/export",
	"languages/available", "languages/list", "languages/add", "languages/update", "languages/delete",
	"terms/list", "terms/add", "terms/update", "terms/delete", "terms/add_comment",
	"translations/add", "translations/update", "translations/delete", "translations/automatic",
	"contributors/list", "contributors/add", "contributors/remove",
}

// destructive endpoints delete data: their raw commands send nothing
// without --yes, like every deleting command. projects/sync deletes every
// term missing from the uploaded list.
var destructive = map[string]bool{
	"projects/delete": true, "projects/sync": true, "languages/delete": true,
	"terms/delete": true, "translations/delete": true, "contributors/remove": true,
}

// groupShort describes the groups that may exist only through raw commands.
var groupShort = map[string]string{
	"projects":     "Work with POEditor projects",
	"languages":    "Work with a project's languages",
	"terms":        "Work with a project's terms",
	"translations": "Work with a project's translations",
	"contributors": "Work with a project's contributors",
}

// AddEndpoints gives every endpoint without a command under apiCmd a raw
// one: `<group> <action>` (dashes for the API's underscores), creating the
// group when it has no dedicated commands yet.
func AddEndpoints(f *cmdutil.Factory, apiCmd *cobra.Command) error {
	for _, endpoint := range Endpoints {
		groupName, action, _ := strings.Cut(endpoint, "/")
		group := child(apiCmd, groupName)
		if group == nil {
			group = cmdutil.NewGroup(groupName+" <command>", groupShort[groupName])
			if err := cmdutil.AddUnique(apiCmd, group); err != nil {
				return err
			}
		}
		if child(group, CommandName(action)) != nil {
			continue // it has a dedicated command
		}
		if err := cmdutil.AddUnique(group, NewCmdEndpoint(f, endpoint)); err != nil {
			return err
		}
	}
	return nil
}

// CommandName is an API action as a command name: add_comment → add-comment.
func CommandName(action string) string { return strings.ReplaceAll(action, "_", "-") }

func child(parent *cobra.Command, name string) *cobra.Command {
	for _, c := range parent.Commands() {
		if c.Name() == name {
			return c
		}
	}
	return nil
}

// request is everything a raw command sends, checked before any API call.
type request struct {
	endpoint string
	params   url.Values
	file     *api.File
}

// NewCmdEndpoint returns the raw command for one endpoint.
func NewCmdEndpoint(f *cmdutil.Factory, endpoint string) *cobra.Command {
	var (
		fields     []string
		fileFields []string
		upload     string
		envelope   bool
		asJSON     bool
		yes        bool
	)
	_, action, _ := strings.Cut(endpoint, "/")

	cmd := &cobra.Command{
		Use:   CommandName(action) + " [--<parameter> <value>]...",
		Short: "Send a " + endpoint + " request (no dedicated command yet)",
		Long: `Send a ` + endpoint + ` request to POEditor API v2 and print the response. The
token of the account (--account, else the default account) is added for you.
See the API documentation for the endpoint's parameters.

API parameters are flags named after them: --id 7717, --id=7717, and
--reference-language en for reference_language. A parameter flag given alone
(--overwrite) sends 1; a value starting with "-" needs the = form (--x=-1).
The command's own flags below keep their meaning; for a parameter that has the
same name as one of them, or whose value should come from a file, use --field
or --field-file.

The "result" of the response is printed like every other command's data: a
table on a terminal (a list becomes rows, an object a single row, nested
values compact JSON), tab-separated without a header when piped, and JSON with
--json, keys in the order the API sent them. --envelope prints the whole
response as JSON, including the "response" status block. When the API refuses
the request, the command exits 1 with the API's error (and with --envelope, the
response is printed first).`,
		// the endpoint's parameters can't be declared as flags: splitArgs
		// separates them from ours, then we parse ours
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.InitDefaultHelpFlag()
			parsed, err := splitArgs(cmd, args)
			if err != nil {
				return err
			}
			// cmd.ParseFlags is a no-op under DisableFlagParsing: parse ours
			// directly, with the inherited ones (--account) merged in
			flags := cmd.Flags()
			flags.AddFlagSet(cmd.InheritedFlags())
			if err := flags.Parse(parsed.ours); err != nil {
				return err
			}
			if help, _ := cmd.Flags().GetBool("help"); help {
				return cmd.Help()
			}
			if len(parsed.positional) > 0 {
				return fmt.Errorf("unexpected argument %q: API parameters are given as --name value", parsed.positional[0])
			}

			req, err := buildRequest(f, endpoint, parsed.params, fields, fileFields, upload)
			if err != nil {
				return err
			}
			if destructive[endpoint] && !yes {
				return fmt.Errorf("%s deletes data and cannot be undone; nothing was sent. To send it, run:\n  %s %s --yes",
					endpoint, cmd.CommandPath(), cmdutil.ShellJoin(args))
			}
			client, err := f.APIClient()
			if err != nil {
				return err
			}

			resp, err := client.Raw(cmd.Context(), req.endpoint, req.params, req.file)
			var apiErr *api.Error
			if err != nil && !errors.As(err, &apiErr) {
				return err // never reached the API, or no envelope came back
			}
			var printErr error
			switch {
			case envelope:
				printErr = printJSON(f, resp.Body)
			case err != nil:
				// refused: nothing to show but the error
			case asJSON:
				printErr = printJSON(f, resp.Result)
			default:
				printErr = printTable(f, resp.Result)
			}
			if printErr != nil {
				return printErr
			}
			return err
		},
	}

	cmd.Flags().StringArrayVarP(&fields, "field", "f", nil, "Add a parameter `key=value` (repeatable; repeat a key to send it several times)")
	cmd.Flags().StringArrayVarP(&fileFields, "field-file", "F", nil, "Add a parameter `key=path` whose value is the file's content (\"-\" = stdin), e.g. -F data=terms.json")
	cmd.Flags().StringVar(&upload, "upload", "", "Attach a file as the multipart field \"file\" (projects upload)")
	cmd.Flags().BoolVar(&envelope, "envelope", false, "Print the whole response as JSON, not only its result")
	cmdutil.AddJSONFlag(cmd, &asJSON)
	if destructive[endpoint] {
		cmd.Long += "\n\n" + endpoint + " deletes data: without --yes nothing is sent."
		cmd.Flags().BoolVar(&yes, "yes", false, "Send the request: it deletes data and cannot be undone")
	}
	return cmd
}

// buildRequest checks the arguments and reads the files; it makes no API call.
// params are the --name value parameters; --field and --field-file add to them.
func buildRequest(f *cmdutil.Factory, endpoint string, params url.Values, fields, fileFields []string, upload string) (request, error) {
	req := request{endpoint: endpoint, params: params}

	for _, field := range fields {
		key, value, err := splitField("--field", field)
		if err != nil {
			return req, err
		}
		req.params.Add(key, value)
	}
	for _, field := range fileFields {
		key, path, err := splitField("--field-file", field)
		if err != nil {
			return req, err
		}
		data, err := f.ReadFile(path)
		if err != nil {
			return req, fmt.Errorf("--field-file %s: %w", key, err)
		}
		req.params.Add(key, string(data))
	}

	if upload != "" {
		data, err := f.ReadFile(upload)
		if err != nil {
			return req, fmt.Errorf("--upload: %w", err)
		}
		req.file = &api.File{Field: "file", Name: filepath.Base(upload), Data: data}
	}
	return req, nil
}

// splitField splits key=value.
func splitField(flag, field string) (string, string, error) {
	key, value, ok := strings.Cut(field, "=")
	if !ok || key == "" {
		return "", "", fmt.Errorf("%s %q: use key=value", flag, field)
	}
	key, err := paramName(flag+" "+key, key)
	return key, value, err
}

// printJSON indents raw JSON as received (key order kept, strings readable);
// nothing to print (an endpoint without a result) prints nothing.
func printJSON(f *cmdutil.Factory, raw json.RawMessage) error {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return nil
	}
	raw, err := readable(raw)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := json.Indent(&buf, raw, "", "  "); err != nil {
		return fmt.Errorf("the API returned invalid JSON: %w", err)
	}
	buf.WriteByte('\n')
	_, err = f.IO.Out.Write(buf.Bytes())
	return err
}
