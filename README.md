<p align="center">
  <a href="https://poeditor.com">
    <picture>
      <source media="(prefers-color-scheme: dark)" srcset="docs/images/poeditor-logo-dark-mode.svg">
      <img src="docs/images/poeditor-logo.svg" alt="POEditor" width="320">
    </picture>
  </a>
</p>

<h3 align="center">POEditor CLI</h3>

<p align="center">
  Your <a href="https://poeditor.com">POEditor</a> projects, terms and translations, from the command line.<br>
  Sync localization files with <code>pull</code> and <code>push</code>, script everything else with <code>poeditor api</code>.
</p>

<p align="center">
  <a href="#install">Install</a> ·
  <a href="#quick-start">Quick start</a> ·
  <a href="#commands">Commands</a> ·
  <a href="#ci">CI</a>
</p>

---

`poeditor` is a single binary for Linux, macOS and Windows. It talks to the public
[POEditor API v2](https://poeditor.com/docs/api) with your API token, and keeps your
repository's language files in step with POEditor:

```console
$ poeditor pull
created        locales/fr.json
updated        locales/de.json
unchanged      locales/en.json
3 files: 1 created, 1 updated, 1 unchanged
```

## Install

Homebrew (macOS and Linux):

```sh
brew install poeditor/tap/poeditor
```

Scoop (Windows):

```powershell
scoop bucket add poeditor https://github.com/POEditor/scoop-bucket
scoop install poeditor
```

Without a package manager, on macOS and Linux:

```sh
curl -fsSL https://raw.githubusercontent.com/POEditor/poeditor-cli/master/install.sh | sh
```

Windows (PowerShell):

```powershell
irm https://raw.githubusercontent.com/POEditor/poeditor-cli/master/install.ps1 | iex
```

The scripts install the latest release after checking it against the release's checksums. Set
`POEDITOR_VERSION` (e.g. `v0.1.0`) to install another release, and `POEDITOR_INSTALL_DIR` to
choose where the binary goes.

Or download the archive for your platform from the
[latest release](https://github.com/POEditor/poeditor-cli/releases/latest), unpack it and put
`poeditor` somewhere on your `PATH`.

With Go 1.27 or later, you can also build it yourself:

```sh
go install github.com/POEditor/poeditor-cli/cmd/poeditor@latest
```

Check it works with `poeditor --version`.

## Quick start

**1. Log in.** Create an API token on your [POEditor account page](https://poeditor.com/account/api),
then:

```sh
poeditor auth login
```

The token is checked, then kept in your system keyring (macOS Keychain, Windows Credential
Manager, libsecret on Linux). More accounts? See [Accounts and tokens](docs/auth.md).

**2. Describe your files.** In the root of your repository:

```sh
poeditor config init
```

A short wizard picks the project, finds your language files (`locales/{lang}.json`,
`values-{lang}/strings.xml`, …) and writes `.poeditor.yml`. Commit it: it holds no secrets.
Everything it can contain is in [`.poeditor.yml`](docs/config.md).

**3. Sync.**

```sh
poeditor push      # upload the source-language files: new and changed terms
poeditor pull      # download every language into its file
```

See [pull and push](docs/pull-push.md) for what each one sends, `--dry-run`, and the
upload limits.

## Commands

| Command | What it does | Docs |
|---|---|---|
| `poeditor pull`, `poeditor push` | Sync the files listed in `.poeditor.yml` | [pull and push](docs/pull-push.md) |
| `poeditor config …` | Create, extend and check `.poeditor.yml` | [`.poeditor.yml`](docs/config.md) |
| `poeditor auth …` | Log in, several accounts, tokens in CI | [Accounts and tokens](docs/auth.md) |
| `poeditor api projects …` | List, create, change, export and upload projects | [Projects](docs/projects.md) |
| `poeditor api languages …` | A project's languages and their progress | [Languages](docs/languages.md) |
| `poeditor api terms …` | List, add, change, delete and comment on terms | [Terms](docs/terms.md) |
| `poeditor api translations …` | Write, delete and machine-translate translations | [Translations](docs/translations.md) |
| `poeditor api contributors …` | Who has access, and to which languages | [Contributors](docs/contributors.md) |

Every POEditor API v2 endpoint is one command, `poeditor api <group> <action>`: the API's
`terms/add_comment` is `poeditor api terms add-comment`. Every command has `--help`.

Commands that work on a project take `--project` (`-p`): its id, or its alias from
`.poeditor.yml`. `--id`, the API's own name for it, works too:

```sh
poeditor api terms list --project web
poeditor api terms list --id 12345
```

### Output

Commands that show data print a table on a terminal, and tab-separated lines without a header
when piped, so `cut` and `awk` work. `--json` prints JSON instead, in a stable shape of our
own:

```sh
poeditor api languages list --project web
poeditor api terms list -p web --language fr --json | jq '.[] | select(.translation == null)'
```

Commands that change something report on stderr, so stdout stays clean for scripts.
Commands that delete anything ask for `--yes` instead of prompting, and without it change
nothing and print the exact command to run.

## CI

No login is needed in CI: set `POEDITOR_TOKEN` (or `POEDITOR_TOKEN_<ACCOUNT>` for a named
account) and run the command.

```yaml
# GitHub Actions
- run: poeditor config check && poeditor pull
  env:
    POEDITOR_TOKEN: ${{ secrets.POEDITOR_TOKEN }}
```

`poeditor config check` exits with an error when `.poeditor.yml` points at a project,
language or account that does not exist. A read-only token is enough for `pull`. More in
[Accounts and tokens](docs/auth.md#ci).

## Help and feedback

- `poeditor <command> --help` for every command and flag.
- [POEditor API v2 reference](https://poeditor.com/docs/api)
- Questions or problems: [contact POEditor](https://poeditor.com/contact/) or open an issue.
