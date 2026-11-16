// Package raw is `poeditor api <group> <action>`: send any API v2 request with the
// account's token and print the response, like `gh api`. It is the escape
// hatch for endpoints that have no command yet.
package raw

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/POEditor/poeditor-cli/internal/api"
	"github.com/POEditor/poeditor-cli/internal/cmdutil"
)

// endpointPattern is <group>/<action>, as the API v2 docs write endpoints.
var endpointPattern = regexp.MustCompile(`^[a-z]+/[a-z_]+$`)

// request is everything `poeditor api` sends, checked before any API call.
type request struct {
	endpoint string
	params   url.Values
	file     *api.File
}

// NewCmdAPI returns `poeditor api`.
func NewCmdAPI(f *cmdutil.Factory) *cobra.Command {
	var (
		fields     []string
		fileFields []string
		upload     string
		envelope   bool
		asJSON     bool
	)

	cmd := &cobra.Command{
		Use:   "api <group> <action> [--<parameter> <value>]...",
		Short: "Send a request to any POEditor API v2 endpoint",
		Long: `Send a request to any POEditor API v2 endpoint and print the response.

The endpoint is given as two words, <group> <action> (projects view, terms
add-comment), or in the API documentation's spelling, projects/view or
terms/add_comment. The token of the account (--account, else the default
account) is added for you.

API parameters are flags named after them: --id 7717, --id=7717, and
--reference-language en for reference_language. A parameter flag given alone
(--overwrite) sends 1; a value starting with "-" needs the = form (--x=-1).
Put parameters after the endpoint. The command's own flags below keep their
meaning; for a parameter that has the same name as one of them, or whose value
should come from a file, use --field or --field-file.

The "result" of the response is printed like every other command's data: a
table on a terminal (a list becomes rows, an object a single row, nested
values compact JSON), tab-separated without a header when piped, and JSON with
--json, keys in the order the API sent them. --envelope prints the whole
response as JSON, including the "response" status block. When the API refuses
the request, the command exits 1 with the API's error (and with --envelope, the
response is printed first).

This is the escape hatch for anything that has no command yet; prefer the
dedicated commands (poeditor projects list…) when they exist.`,
		Example: `  poeditor api projects list
  poeditor api projects view --id 7717
  poeditor api projects update --id 7717 --reference-language en
  poeditor api terms add --id 7717 -F data=terms.json
  poeditor api projects upload --id 7717 --updating terms --overwrite --upload en.json
  poeditor api languages list --id=7717 --json
  poeditor api terms/add_comment --id 7717 -F data=comments.json`,
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
			if n := len(parsed.positional); n < 1 || n > 2 {
				return fmt.Errorf("expected <group> <action> (e.g. `poeditor api projects list`), got %d argument(s)", n)
			}

			req, err := buildRequest(f, endpoint(parsed.positional), parsed.params, fields, fileFields, upload)
			if err != nil {
				return err
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
	cmd.Flags().StringVar(&upload, "upload", "", "Attach a file as the multipart field \"file\" (projects/upload)")
	cmd.Flags().BoolVar(&envelope, "envelope", false, "Print the whole response as JSON, not only its result")
	cmdutil.AddJSONFlag(cmd, &asJSON)
	return cmd
}

// endpoint joins `<group> <action>` into the API's <group>/<action>, with
// dashes as the API's underscores (add-comment → add_comment).
func endpoint(args []string) string {
	if len(args) == 1 {
		return args[0]
	}
	return args[0] + "/" + strings.ReplaceAll(args[1], "-", "_")
}

// buildRequest checks the arguments and reads the files; it makes no API call.
// params are the --name value parameters; --field and --field-file add to them.
func buildRequest(f *cmdutil.Factory, endpoint string, params url.Values, fields, fileFields []string, upload string) (request, error) {
	req := request{endpoint: strings.Trim(endpoint, "/"), params: params}
	if !endpointPattern.MatchString(req.endpoint) {
		return req, fmt.Errorf("invalid endpoint %q: use <group> <action>, e.g. `poeditor api projects list`", endpoint)
	}

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
		data, err := readFile(f, path)
		if err != nil {
			return req, fmt.Errorf("--field-file %s: %w", key, err)
		}
		req.params.Add(key, string(data))
	}

	if upload != "" {
		data, err := readFile(f, upload)
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

// readFile reads path (relative to the working directory), or stdin for "-".
func readFile(f *cmdutil.Factory, path string) ([]byte, error) {
	if path == "-" {
		return io.ReadAll(f.IO.In)
	}
	if !filepath.IsAbs(path) {
		dir, err := f.WorkingDir()
		if err != nil {
			return nil, err
		}
		path = filepath.Join(dir, path)
	}
	return os.ReadFile(path)
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
