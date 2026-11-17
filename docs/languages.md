# Languages

`poeditor api languages` shows a project's languages and how far each is translated, adds and
removes languages, and writes many translations of one language at once.

Languages are always given by their POEditor code: `en`, `fr`, `pt-br`, `zh-Hans`…
[`poeditor api languages available`](#poeditor-api-languages-available) lists them all. A wrong
code is explained: either POEditor doesn't know it, or the project doesn't have it, and then
the error lists the languages the project has.

Every command here except `available` takes `--project` (`-p`, or `--id` as in the API docs),
an id or an alias from `.poeditor.yml`. See [Picking a project](projects.md#picking-a-project).

[← back to the README](../README.md)

## `poeditor api languages list`

A project's languages: code, name, how many terms are translated, the progress, and when a
translation was last added.

```console
$ poeditor api languages list --project web
CODE   NAME             TRANSLATIONS  PROGRESS  UPDATED
fr     French           10            83.33%    2026-03-01
                                                08:00:00

pt-br  Portuguese (BR)  6             50%       2026-01-14
                                                09:12:44

ro     Romanian         0             0%
```

Piped, the progress is a bare number (`83.33`). `--json` prints the list as JSON, with
`percentage` as a number.

## `poeditor api languages available`

Every language you can add to a project: its code and its name. The account's custom languages
are included, so the list depends on `--account`. It takes no `--project`.

```sh
poeditor api languages available
poeditor api languages available | grep -i portuguese
```

`--json` prints the list as JSON.

## `poeditor api languages add`

Add one or more languages to a project.

```sh
poeditor api languages add fr --project web
poeditor api languages add de es pt-br -p 12345
```

A language the project already has is left alone, with a note on stderr. The first code
POEditor refuses stops the command; the languages added before it stay added.

## `poeditor api languages delete`

Remove a language from a project, with **all its translations**. This cannot be undone, so it
needs `--yes`: without it nothing is deleted and the command prints the line to run.

```sh
poeditor api languages delete fr --project web --yes
```

## `poeditor api languages update`

Write many translations of one language at once, from a JSON file (`--from-file`, or `-` for
stdin). Existing translations are **overwritten**, and terms that are not in the project are
skipped.

```sh
poeditor api languages update fr --project web --from-file fr.json
jq '…' export.json | poeditor api languages update fr -p 12345 --from-file -
poeditor api languages update fr -p web --from-file fr.json --fuzzy-trigger
```

| Flag | |
|---|---|
| `--from-file <file>` | The translations: a JSON file, or `-` for stdin (required). |
| `--fuzzy-trigger` | Mark the other languages' translations fuzzy where this update added or changed one, so they get reviewed. |
| `--json` | Print the counts (parsed, added, updated) as JSON, as well as on stderr. |

The file is a list of terms and their translation, in the API's shape:

```json
[
  {"term": "save", "context": "button",
   "translation": {"content": "Enregistrer", "fuzzy": 0}},
  {"term": "files",
   "translation": {"content": {"one": "1 fichier", "other": "%d fichiers"}}}
]
```

- `context` may be left out (the term has none), and so may `fuzzy` (`0`). `fuzzy: 1` marks the
  translation fuzzy, for review.
- A plural term's `content` is an object of its forms (`one`, `few`, `many`, `other`…), kept in
  your order.
- The file is checked before anything is sent: an empty list, an entry without a `term`, or a
  misspelled field (`"contents"`) is refused, naming the entry.

To fill only empty translations, or to overwrite only existing ones, see
[translations add and update](translations.md).
