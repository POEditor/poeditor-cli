# Translations

`poeditor api translations` writes and deletes the translations of terms in one language, and
machine-translates a project with Google, Microsoft or DeepL.

Every command here takes `--project` (`-p`, or `--id` as in the API docs), an id or an alias
from `.poeditor.yml`. See [Picking a project](projects.md#picking-a-project).

[← back to the README](../README.md)

## Which command writes what

| Command | Empty translation | Existing translation |
|---|---|---|
| `translations add` | filled | left as it is |
| `translations update` | left empty | overwritten |
| [`languages update`](languages.md#poeditor-api-languages-update) | filled | overwritten |

`add` is safe to repeat: it never touches work that is already done. `update` only fixes what
exists. `languages update` writes everything, and is also the only one that can mark
translations fuzzy.

## One translation, or many

`add`, `update` and `delete` take **one term** from their arguments and flags, or **many** from
a JSON file with `--from-file` (`-` reads stdin), always in one `--language`. A term is found by
its text and its `--context`; leave `--context` out for a term without one.

The file is checked before anything is sent: an empty list, an entry without `term` or
`content`, an entry with `fuzzy` set (these endpoints ignore it), or a field the command doesn't
know is refused, naming the entry.

## `poeditor api translations add`

Add the translation of a term where it has none, or of many terms. A term that already has a
translation keeps it.

```sh
poeditor api translations add save Enregistrer --project web --language fr --context button
poeditor api translations add "%d files" -p 12345 -l fr --plural-form one="1 fichier" --plural-form other="%d fichiers"
poeditor api translations add -p web -l fr --from-file fr.json
```

| Flag | |
|---|---|
| `--language <code>` (`-l`) | The language (required). |
| `--context <context>` | The term's context. |
| `--plural-form <form>=<text>` | For a plural term, instead of the text: one per form (`one`, `few`, `many`, `other`…), in order. |
| `--from-file <file>` | Add the translations of a JSON file instead. |
| `--json` | Print the counts (parsed, added) as JSON. |

```json
[
  {"term": "save", "context": "button", "translation": {"content": "Enregistrer"}},
  {"term": "%d files", "translation": {"content": {"one": "1 fichier", "other": "%d fichiers"}}}
]
```

The summary counts the translations you sent, and says why some weren't added: the term
already had one, or isn't in the project.

## `poeditor api translations update`

Overwrite the translation of a term, or of many. A term with no translation yet is skipped.
The flags and the file are the same as for [`add`](#poeditor-api-translations-add).

```sh
poeditor api translations update save Enregistrer --project web --language fr --context button
poeditor api translations update -p web -l fr --from-file fixes.json
```

| Flag | |
|---|---|
| `--language <code>` (`-l`) | The language (required). |
| `--context <context>` | The term's context. |
| `--plural-form <form>=<text>` | For a plural term, one per form, instead of the text. |
| `--from-file <file>` | Apply the translations of a JSON file instead. |
| `--json` | Print the counts (parsed, updated) as JSON. |

## `poeditor api translations delete`

Delete the translation of a term in one language, or of many terms. The terms, and their other
languages, stay. This cannot be undone, so it needs `--yes`: without it nothing is deleted and
the command prints the line to run.

```sh
poeditor api translations delete save --project web --language fr --context button --yes
poeditor api translations delete -p 12345 -l fr --from-file stale.json --yes
```

| Flag | |
|---|---|
| `--language <code>` (`-l`) | The language (required). |
| `--context <context>` | The term's context. |
| `--from-file <file>` | Delete the translations of the terms listed in a JSON file instead: `[{"term": "save", "context": "button"}]`. |
| `--yes` | Delete. |
| `--json` | Print the counts (parsed, deleted) as JSON. |

## `poeditor api translations automatic`

Machine-translate a project from one of its languages into others, using the account's
automatic translation characters. Only empty translations are filled.

```sh
poeditor api translations automatic --project web --provider deepl --source-language en --target fr --target de
poeditor api translations automatic -p 12345 --provider google --source-language en --target pt-br=pt --tag new
poeditor api translations automatic -p web --provider microsoft --source-language en --target ro --option format=html --json
```

| Flag | |
|---|---|
| `--provider google\|microsoft\|deepl` | Who translates (required). |
| `--source-language <code>[=<provider code>]` | The project language to translate from (required). |
| `--target <code>[=<provider code>]` | A language to translate to (repeat; required). |
| `--tag <tag>` | Only the terms with this tag (repeat for more). |
| `--tags-match all\|any\|exclude` | How `--tag` matches: every tag, at least one (the default), or none of them. |
| `--option <key>=<value>` | A provider option, such as `format=html` or `formality=less` (repeat). The [API documentation](https://poeditor.com/docs/api#translations_automatic) lists them per provider. |
| `--json` | Print the result as JSON. |

**Language codes.** Languages are the project's codes. When the provider spells one
differently, add its code after `=`: `--target pt-br=pt-BR`. Without it, the project's code is
sent to the provider as it is.

**Result.** One line per language pair: the characters translated, or why it failed.

```console
$ poeditor api translations automatic -p web --provider google --source-language en --target fr --target ar
SOURCE  TARGET  CHARACTERS  ERROR
en      fr      1204
en      ar                  Not enough automatic translation characters.
error: 1 of 2 language pairs failed
```

When any pair fails, the command exits with an error after printing them all, so a CI job
notices. A pair with nothing left to translate (`No untranslated terms found.`) is
printed but does not count as failed, so running it on every push is safe.
