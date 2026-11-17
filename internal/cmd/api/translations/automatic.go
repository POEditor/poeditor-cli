package translations

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/POEditor/poeditor-cli/internal/api"
	"github.com/POEditor/poeditor-cli/internal/cmdutil"
	"github.com/POEditor/poeditor-cli/internal/projectflag"
	"github.com/POEditor/poeditor-cli/internal/repoconfig"
)

// NewCmdAutomatic returns `poeditor api translations automatic` (API v2
// translations/automatic).
func NewCmdAutomatic(f *cmdutil.Factory) *cobra.Command {
	var (
		project   string
		provider  string
		source    string
		targets   []string
		tags      []string
		tagsMatch string
		options   []string
		asJSON    bool
	)

	cmd := &cobra.Command{
		Use:   "automatic",
		Short: "Machine-translate a project's languages (Google, Microsoft, DeepL)",
		Long: `Machine-translate a project from one of its languages into others, with the
account's automatic translation characters. Only empty translations are
filled.

Languages are the project's codes. When the provider names a language
differently, add its code after "=": --target pt-br=pt-BR. --tag limits the
terms to translate (with --tags-match: all, any or exclude); --option passes
a provider option (see the API documentation), e.g. --option format=html.

Each language pair is one line: the characters translated, or why it failed.
When a pair fails, the command exits 1 after printing them all.`,
		Example: `  poeditor api translations automatic --project web --provider deepl --source-language en --target fr --target de
  poeditor api translations automatic -p 7717 --provider google --source-language en --target pt-br=pt --tag new
  poeditor api translations automatic -p web --provider microsoft --source-language en --target ro --option format=html --json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts, err := automaticOptions(provider, source, targets, tags, tagsMatch, options)
			if err != nil {
				return err
			}
			p, err := projectflag.Resolve(f, project)
			if err != nil {
				return err
			}
			result, err := p.Client.TranslationsAutomatic(cmd.Context(), p.ID, opts)
			if err != nil {
				return err
			}
			if asJSON {
				if err := cmdutil.PrintJSON(f.IO, automaticJSON(p.Ref, provider, result)); err != nil {
					return err
				}
			} else if err := printPairs(f, result); err != nil {
				return err
			}
			if n := len(result.Failed); n > 0 {
				return fmt.Errorf("%d of %d language pairs failed", n, n+len(result.Successful))
			}
			return nil
		},
	}

	projectflag.Add(cmd, &project)
	cmd.Flags().StringVar(&provider, "provider", "", "The `provider`: "+strings.Join(api.Providers, ", ")+" (required)")
	cmd.Flags().StringVar(&source, "source-language", "", "The `language` to translate from, as code[=provider code] (required)")
	cmd.Flags().StringArrayVar(&targets, "target", nil, "A `language` to translate to, as code[=provider code] (repeat; required)")
	cmd.Flags().StringArrayVar(&tags, "tag", nil, "Only the terms with this `tag` (repeat for more)")
	cmd.Flags().StringVar(&tagsMatch, "tags-match", "", "How --tag matches (`mode`: "+strings.Join(api.TagsMatchModes, ", ")+"; the API's default: any)")
	cmd.Flags().StringArrayVar(&options, "option", nil, "A provider option, `key=value` (repeat)")
	_ = cmd.MarkFlagRequired("provider")
	_ = cmd.MarkFlagRequired("source-language")
	_ = cmd.MarkFlagRequired("target")
	cmdutil.AddJSONFlag(cmd, &asJSON)
	return cmd
}

// automaticOptions maps the flags onto the API's parameters, checking them
// before any API call.
func automaticOptions(provider, source string, targets, tags []string, tagsMatch string, options []string) (api.AutomaticOptions, error) {
	o := api.AutomaticOptions{Provider: provider, Tags: tags, TagsMatch: tagsMatch}
	if !slices.Contains(api.Providers, provider) {
		return o, fmt.Errorf("--provider %q: use %s", provider, strings.Join(api.Providers, ", "))
	}
	var err error
	if o.SourceLanguage, o.ProviderSourceLanguage, err = languagePair("--source-language", source); err != nil {
		return o, err
	}
	for _, t := range targets {
		code, providerCode, err := languagePair("--target", t)
		if err != nil {
			return o, err
		}
		o.Targets = append(o.Targets, api.AutomaticTarget{ProjectLanguage: code, ProviderLanguage: providerCode})
	}
	if tagsMatch != "" {
		if !slices.Contains(api.TagsMatchModes, tagsMatch) {
			return o, fmt.Errorf("--tags-match %q: use %s", tagsMatch, strings.Join(api.TagsMatchModes, ", "))
		}
		if len(tags) == 0 {
			return o, fmt.Errorf("--tags-match needs --tag")
		}
	}
	for _, opt := range options {
		key, value, ok := strings.Cut(opt, "=")
		if !ok || strings.TrimSpace(key) == "" {
			return o, fmt.Errorf("--option %q: use key=value", opt)
		}
		if key == "tags" || key == "tag" || key == "tags_match" {
			return o, fmt.Errorf("--option %s: use --tag and --tags-match", key)
		}
		if o.Options == nil {
			o.Options = map[string]string{}
		}
		o.Options[key] = value
	}
	return o, nil
}

// languagePair reads code[=provider code]; the provider code defaults to
// the project's.
func languagePair(flag, value string) (code, providerCode string, err error) {
	code, providerCode, found := strings.Cut(value, "=")
	code, providerCode = strings.TrimSpace(code), strings.TrimSpace(providerCode)
	if code == "" || (found && providerCode == "") {
		return "", "", fmt.Errorf("%s %q: use a language code, or code=provider-code (e.g. pt-br=pt-BR)", flag, value)
	}
	if !found {
		providerCode = code
	}
	return code, providerCode, nil
}

// printPairs is one line per language pair: the successful ones, then the
// failed ones with the API's reason.
func printPairs(f *cmdutil.Factory, r api.AutomaticResult) error {
	table := cmdutil.NewTable(f.IO, "SOURCE", "TARGET", "CHARACTERS", "ERROR")
	for _, pr := range r.Successful {
		table.AddRow(pr.Source, pr.Target, strconv.Itoa(pr.Chars), "")
	}
	for _, pr := range r.Failed {
		table.AddRow(pr.Source, pr.Target, "", pr.Message)
	}
	return table.Render()
}

// automaticResult is what --json prints: our own shape.
type automaticResult struct {
	Project    repoconfig.ProjectRef `json:"project"`
	Provider   string                `json:"provider"`
	Successful []translatedPair      `json:"successful"`
	Failed     []failedPair          `json:"failed"`
}

type translatedPair struct {
	Source     string `json:"source"`
	Target     string `json:"target"`
	Characters int    `json:"characters"`
}

type failedPair struct {
	Source  string `json:"source"`
	Target  string `json:"target"`
	Message string `json:"message"`
}

func automaticJSON(ref repoconfig.ProjectRef, provider string, r api.AutomaticResult) automaticResult {
	out := automaticResult{Project: ref, Provider: provider, Successful: []translatedPair{}, Failed: []failedPair{}}
	for _, pr := range r.Successful {
		out.Successful = append(out.Successful, translatedPair{Source: pr.Source, Target: pr.Target, Characters: pr.Chars})
	}
	for _, pr := range r.Failed {
		out.Failed = append(out.Failed, failedPair{Source: pr.Source, Target: pr.Target, Message: pr.Message})
	}
	return out
}
