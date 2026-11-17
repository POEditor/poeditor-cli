# Accounts and tokens

`poeditor` works with POEditor API tokens: no browser login, no OAuth. Create a token on your
[POEditor account page](https://poeditor.com/account/api) and give it to `poeditor auth login`.

- Tokens are kept in your system keyring: macOS Keychain, Windows Credential Manager, or
  libsecret on Linux. Only where no keyring exists (a headless server) does the token go into
  the user config file, readable by you only.
- A token never goes into `.poeditor.yml`, which you commit.
- In CI, tokens come from environment variables and nothing is stored: see [CI](#ci).

[← back to the README](../README.md)

## Several accounts

If your projects live in more than one POEditor account (yours and a client's, say), give each
one a local name:

```sh
poeditor auth login                      # the first account is called "default"
poeditor auth login --account client-a   # every further one needs a name
```

Names are lower-case letters, digits, `-` and `_`. One account is the default; every command
uses it unless told otherwise:

- `--account <name>` on any command picks the account for that run;
- in `.poeditor.yml`, a project's `account:` key names the account its commands use (see
  [`.poeditor.yml`](config.md));
- `poeditor auth default <name>` changes the default.

There is no guessing: if a project names an account that isn't logged in, the command says
which `poeditor auth login --account …` to run.

## `poeditor auth login`

Store an API token for an account. The token is checked against the API before anything is
saved (the command reports how many projects it can see), then stored in the keyring. Logging
in again to an existing account replaces its token.

```sh
poeditor auth login
poeditor auth login --account client-a
echo "$TOKEN" | poeditor auth login --account ci --with-token
```

On a terminal the token is asked for with hidden input. `--with-token` reads it from standard
input instead, with no prompt, for scripts.

The first account is called `default` unless you name it with `--account`. Once one exists,
`login` without `--account` is refused rather than overwriting it.

## `poeditor auth list`

List your accounts: which one is the default, and where each token comes from (`keyring`,
`config file`, `environment`, or `missing`).

```console
$ poeditor auth list
ACCOUNT   DEFAULT  TOKEN
client-a           keyring
personal  yes      keyring
```

`--json` prints the same as JSON.

## `poeditor auth default`

Make an account the default, the one used when no `--account` is given and a project names
none.

```sh
poeditor auth default client-a
```

## `poeditor auth logout`

Remove an account and its stored token. With a single account no flag is needed; with several,
name the one to remove with `--account`. If it was the default, the first remaining account,
alphabetically, becomes the default.

```sh
poeditor auth logout --account client-a
```

## CI

In CI, set the token in the environment instead of logging in:

| Variable | Token for |
|---|---|
| `POEDITOR_TOKEN` | the default account, or the only one when no account is configured at all |
| `POEDITOR_TOKEN_<ACCOUNT>` | the account `<account>`, upper-cased, `-` as `_`: `client-a` → `POEDITOR_TOKEN_CLIENT_A` |

Environment variables win over the keyring and the config file, so a CI job never runs `auth
login`.

```yaml
# GitHub Actions
- run: poeditor pull
  env:
    POEDITOR_TOKEN: ${{ secrets.POEDITOR_TOKEN }}
    POEDITOR_TOKEN_CLIENT_A: ${{ secrets.POEDITOR_TOKEN_CLIENT_A }}
```

**Least privilege:** POEditor tokens can be read-only. A job that only runs `pull` or
`config check` can use one; `push` and anything that changes POEditor need a full-access
token.

## Other environment variables

| Variable | Effect |
|---|---|
| `POEDITOR_CONFIG_DIR` | Where the user config lives (default `~/.config/poeditor`, or `%AppData%\POEditor` on Windows) |
| `POEDITOR_FORCE_TTY=1` | Treat stdout as a terminal (tables with headers), e.g. in Git Bash on Windows, which reports a pipe |
| `POEDITOR_API_URL` | Another API base URL, for testing against a POEditor server of your own |
