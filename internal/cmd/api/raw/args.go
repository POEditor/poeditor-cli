package raw

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// parsedArgs is the command line of `poeditor api`, split by splitArgs.
type parsedArgs struct {
	ours       []string   // our own flags, for cmd.ParseFlags
	positional []string   // <group> <action>, or <group>/<action>
	params     url.Values // every other --name value: an API parameter
}

// splitArgs separates the command line, since cobra cannot: an endpoint's
// parameters are not known in advance, so they can't be declared as flags.
//
//   - A flag the command knows (--json, --envelope, --upload, -f, -F, the
//     global --account, --help) keeps its meaning.
//   - Any other --name becomes the API parameter name (dashes → underscores):
//     --name value, --name=value, or --name alone, which sends 1 (API v2
//     reads 1 as yes). A value starting with "-" needs the = form.
//   - Everything after "--" is positional.
func splitArgs(cmd *cobra.Command, args []string) (parsedArgs, error) {
	parsed := parsedArgs{params: url.Values{}}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--":
			parsed.positional = append(parsed.positional, args[i+1:]...)
			return parsed, nil

		case strings.HasPrefix(arg, "--"):
			name, value, hasValue := strings.Cut(arg[2:], "=")
			if flag := lookup(cmd, name); flag != nil {
				parsed.ours = append(parsed.ours, arg)
				if !hasValue && takesValue(flag) && i+1 < len(args) {
					i++
					parsed.ours = append(parsed.ours, args[i])
				}
				continue
			}
			if !hasValue {
				value = "1"
				if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
					i++
					value = args[i]
				}
			}
			key, err := paramName(arg, strings.ReplaceAll(name, "-", "_"))
			if err != nil {
				return parsed, err
			}
			parsed.params.Add(key, value)

		case strings.HasPrefix(arg, "-") && len(arg) > 1:
			// shorthands are only ours: -f, -F, -h
			flag := cmd.Flags().ShorthandLookup(arg[1:2])
			if flag == nil {
				flag = cmd.InheritedFlags().ShorthandLookup(arg[1:2])
			}
			if flag == nil {
				return parsed, fmt.Errorf("unknown shorthand flag %q: API parameters are given as --name value", arg)
			}
			parsed.ours = append(parsed.ours, arg)
			if len(arg) == 2 && takesValue(flag) && i+1 < len(args) {
				i++
				parsed.ours = append(parsed.ours, args[i])
			}

		default:
			parsed.positional = append(parsed.positional, arg)
		}
	}
	return parsed, nil
}

// lookup finds one of our flags, local or inherited (--account, --help).
func lookup(cmd *cobra.Command, name string) *pflag.Flag {
	if flag := cmd.Flags().Lookup(name); flag != nil {
		return flag
	}
	return cmd.InheritedFlags().Lookup(name)
}

// takesValue is false for boolean flags (--json), which take no argument.
func takesValue(flag *pflag.Flag) bool { return flag.NoOptDefVal == "" }

// paramName checks an API parameter name; the token never is one.
func paramName(arg, key string) (string, error) {
	if key == "" {
		return "", fmt.Errorf("%s: missing parameter name", arg)
	}
	if key == "api_token" {
		return "", fmt.Errorf("%s: the token comes from the account; choose it with --account", arg)
	}
	return key, nil
}
