# Projects

`poeditor api projects` lists, creates, changes and deletes POEditor projects, and moves single
files in and out of them.

[← back to the README](../README.md)

## Picking a project

Every command that works on one project takes `--project` (`-p`), always required, so a command
never acts on the wrong project by accident:

- a **project id**: `--project 12345` works anywhere;
- an **alias** from [`.poeditor.yml`](config.md): `--project web`. The project's `account:`
  from the file is used too.

`--id` works everywhere `--project` does: it is the API documentation's name for the
parameter, so `--id 12345` copied from there just works. Every command also takes the global
`--account` (see [Accounts and tokens](auth.md)).

## `poeditor api projects list`

List every project the account's token can access: id, name, public and open flags, creation
date.

```console
$ poeditor api projects list
ID     NAME        PUBLIC  OPEN  CREATED
12345  Web         no      no    2026-01-14
                                 09:12:44

67890  Mobile app  yes     yes   2025-06-02
                                 17:03:10
```

With several accounts logged in, a line on stderr says which one you are looking at. `--json`
prints the list as JSON.

## `poeditor api projects view`

Show one project, as one row: name, description, reference and fallback languages, public and
open flags, how many terms it has, when terms were last added and updated (`T ADDED`,
`T UPDATED`), and when the project was created.

```console
$ poeditor api projects view --project web
ID     NAME  DESCRIPTION         REFERENCE  FALLBACK  PUBLIC  OPEN  TERMS  T ADDED     T UPDATED   CREATED
12345  Web   The marketing site  en                   no      no    412    2026-03-01  2026-03-02  2026-01-14
                                                                           08:00:00    09:30:00    09:12:44
```

Piped, it prints one tab-separated line, with dates in the API's format. `--json` prints the
project as JSON.

## `poeditor api projects add`

Create a project owned by the account. It starts with no languages: add them with
[`poeditor api languages add`](languages.md#poeditor-api-languages-add).

```sh
poeditor api projects add "Mobile app"
poeditor api projects add --name Web --description "The marketing site" --json
```

The name is the argument or `--name`, as in `projects update`; giving both is an error.

| Flag | |
|---|---|
| `--name <name>` | The project's name, instead of the argument. |
| `--description <text>` | The project's description. |
| `--json` | Print the new project as JSON. Its id is always on stderr. |

## `poeditor api projects update`

Change a project's settings. Only the flags you give change; giving none is an error.

```sh
poeditor api projects update --project web --name "Web (2026)"
poeditor api projects update -p 12345 --reference-language en --fallback-language ""
poeditor api projects update -p web --proofreading on --lock-proofread on
```

| Flag | |
|---|---|
| `--name <name>` | A new name. |
| `--description <text>` | A new description; `""` clears it. |
| `--reference-language <code>` | The reference language, shown to translators next to theirs; `""` clears it. |
| `--fallback-language <code>` | The language exports fall back to for untranslated terms; `""` clears it. |
| `--proofreading on\|off` | Turn proofreading on or off. |
| `--lock-proofread on\|off` | Keep contributors from changing translations that were proofread. |
| `--json` | Print the project as it is after the change. |

A language must be one of the project's. The command checks it first and lists the project's
languages if not, because POEditor would silently ignore a wrong code and report success.

## `poeditor api projects delete`

Delete a project the account owns, with all its terms, translations, languages and
contributors. This cannot be undone, so it needs `--yes`: without it nothing is deleted and the
command prints the line to run.

```sh
poeditor api projects delete --project 12345 --yes
```

## `poeditor api projects export`

Export one language of a project as a file, and print it, or write it with `--output`.

```sh
poeditor api projects export --project web --language fr --format po > fr.po
poeditor api projects export -p 12345 -l de --format key_value_json -o locales/de.json
poeditor api projects export -p web -l fr --format po --filter translated --tag v2 --json
```

| Flag | |
|---|---|
| `--language <code>` (`-l`) | The language to export (required). |
| `--format <format>` | The file format, by its API name: `po`, `key_value_json`, `apple_strings`, `xliff`… (required; the full list is in [`.poeditor.yml`](config.md#files)). |
| `--filter <filter>` | Only the terms that are `translated`, `untranslated`, `fuzzy`, `not_fuzzy`, `automatic`, `not_automatic`, `ai`, `not_ai`, `tm`, `not_tm`, `proofread` or `not_proofread`. Repeat: all must match. |
| `--tag <tag>` | Only the terms with this tag (repeat for more). |
| `--tags-match all\|any\|exclude` | How `--tag` matches: every tag, at least one (the default), or none of them. |
| `--order terms` | Sort the terms alphabetically. |
| `--fallback-language <code>` | Fill untranslated terms from this language. |
| `--output <path>` (`-o`) | Write the file there instead of stdout. The write is atomic, and a file that would not change is left alone. |
| `--json` | Describe the export instead of printing the file: its download link and, with `--output`, the file written and whether it was `created`, `updated` or `unchanged`. Without `--output`, nothing is downloaded. |

To download every file of `.poeditor.yml`, use [`poeditor pull`](pull-push.md#poeditor-pull).

## `poeditor api projects upload`

Upload one file to a project. POEditor reads the format from the file.

```sh
poeditor api projects upload en.json --project web --updating terms_translations --language en
poeditor api projects upload fr.po -p 12345 --updating translations -l fr --overwrite
poeditor api projects upload en.json -p web --updating terms --sync-terms --tag-new v2 --yes
```

| Flag | |
|---|---|
| `--updating <what>` | What the file brings: `terms` (new terms only), `terms_translations` (terms and their translations in `--language`) or `translations` (translations only, never new terms). Required. |
| `--language <code>` (`-l`) | The language of the file's translations. Required unless `--updating terms`. |
| `--overwrite` | Replace existing translations. Without it, only empty ones are filled. |
| `--sync-terms` | **Delete** every term of the project that is not in the file, with its translations. Needs `--yes`. |
| `--yes` | Confirm `--sync-terms`. Without it, nothing is sent and the command prints the line to run. |
| `--read-from-source` | Read the translations from the file's source text (xliff). |
| `--fuzzy-trigger` | Mark the other languages' translations fuzzy where the source text changed. |
| `--tag-all <tag>` | Tag every term of the file (repeat). |
| `--tag-new <tag>` | Tag the terms the file adds (repeat). |
| `--tag-obsolete <tag>` | Tag the project's terms that are not in the file (repeat). |
| `--tag-overwritten <tag>` | Tag the terms whose translation was overwritten (repeat). |
| `--json` | Print the counts as JSON, in the same shape as `push`. |

The counts (terms parsed, added, deleted; translations added, updated) go to stderr. POEditor
accepts one upload every 10 seconds (20 on free plans); when it says it is too early, the
command waits and retries, like `push`.

To upload the files of `.poeditor.yml`, use [`poeditor push`](pull-push.md#poeditor-push).

## `poeditor api projects sync`

Make a project's terms **exactly** the list in a JSON file: new terms are added, existing ones
get the file's reference, plural, tags and comment, and **every other term is deleted**, with
its translations. It needs `--yes`: without it nothing is sent and the command prints the line
to run.

```sh
poeditor api projects sync terms.json --project web --yes
```

The file (or `-` for stdin) is a list in the API's shape, where only `term` is required:

```json
[
  {"term": "save", "context": "button", "reference": "src/save.js:12", "tags": ["web"]},
  {"term": "%d files", "plural": "%d files", "comment": "Shown in the file list"}
]
```

Every entry must have a `term`, and the command checks this before sending. POEditor deletes
the old terms before it looks at the list, so a file of entries without `term` would empty the
project. `--json` prints the counts (parsed, added, updated, deleted).

To add terms without deleting any, use
[`poeditor api terms add`](terms.md#poeditor-api-terms-add).
