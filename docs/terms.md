# Terms

`poeditor api terms` lists a project's terms (with their translations, if you like), and adds,
changes, deletes and comments on them, one at a time or many at once.

A term is identified by its **text and its context**: `save` with context `button` and `save`
with no context are two terms. Every command that finds a term takes `--context`; leave it out
for a term without context.

Every command here takes `--project` (`-p`, or `--id` as in the API docs), an id or an alias
from `.poeditor.yml`. See [Picking a project](projects.md#picking-a-project).

[← back to the README](../README.md)

## One term, or many

`add`, `update`, `delete` and `add-comment` take **one term** from their arguments and flags,
or **many** from a JSON file with `--from-file` (`-` reads stdin). The file is a list in the
API's shape, shown under each command. You give one or the other, never both: with
`--from-file`, a term's own flags (`--context`, `--tag`…) go into each entry instead.

The file is checked before anything is sent: an empty list, an entry without `term`, or a field
the command doesn't know (a typo like `"tag"` for `"tags"`) is refused, naming the entry.

Each command reports its counts on stderr, with a note when some terms weren't affected
(already there, or not found), and `--json` prints them on stdout:

```console
$ poeditor api terms add -p web --from-file new-terms.json
Added 3 of 4 terms to project web.
1 term already in the project.
```

## `poeditor api terms list`

Every term of a project, with all its fields. With `--language`, each term's translation in that
language is added: on a terminal, its status (fuzzy, proofread, last update) on one line and
the text below it, one line per plural form.

```sh
poeditor api terms list --project web
poeditor api terms list -p 12345 --language fr
poeditor api terms list -p web -l fr --json
```

| Flag | |
|---|---|
| `--language <code>` (`-l`) | Add each term's translation in this language. |
| `--json` | Print the terms as JSON, with everything the API returns. |

Long texts wrap within their column. Piped, each term is one tab-separated line, with a plural
translation as a JSON object.

## `poeditor api terms add`

Add a term, or many. A term already in the project (same text and context) is left as it is.

```sh
poeditor api terms add save --project web --context button --tag web --tag v2
poeditor api terms add "%d file" -p 12345 --plural "%d files"
poeditor api terms add -p web --from-file new-terms.json
```

| Flag | |
|---|---|
| `--context <context>` | The term's context. |
| `--plural <form>` | The term's plural form, which makes it a plural term. |
| `--reference <text>` | Where the term is used, e.g. `src/save.js:12`. |
| `--comment <text>` | A comment for translators. |
| `--tag <tag>` | A tag (repeat for more). |
| `--from-file <file>` | Add the terms of a JSON file instead. |
| `--json` | Print the counts (parsed, added) as JSON. |

```json
[
  {"term": "save", "context": "button", "reference": "src/save.js:12",
   "comment": "Max 10 characters", "tags": ["web", "v2"]},
  {"term": "%d file", "plural": "%d files"}
]
```

## `poeditor api terms update`

Change a term, found by its text and `--context`, or many. Only what you give changes; giving
nothing to change is an error.

```sh
poeditor api terms update save --project web --context button --new-term Save
poeditor api terms update welcome -p 12345 --tag web --tag v2
poeditor api terms update -p web --from-file renames.json --fuzzy-trigger
```

| Flag | |
|---|---|
| `--context <context>` | The term's context, which with its text finds the term. |
| `--new-term <text>` | Change the term's text. |
| `--new-context <context>` | Change the term's context. |
| `--plural <form>` | Set the plural form; `""` makes it a normal term. |
| `--reference <text>` | Set where the term is used; `""` clears it. |
| `--tag <tag>` | **Replace** the term's tags with these (repeat for more). |
| `--fuzzy-trigger` | Mark the updated terms' translations fuzzy, so they get reviewed. Works with `--from-file` too. |
| `--from-file <file>` | Apply the updates of a JSON file instead. |
| `--json` | Print the counts (parsed, updated) as JSON. |

In the file, `term` and `context` find the term, and the other fields are optional. `"tags": []`
removes every tag:

```json
[
  {"term": "save", "context": "button", "new_term": "Save",
   "reference": "src/save.js:14", "tags": ["web"]},
  {"term": "old_key", "new_term": "new_key", "new_context": "home"},
  {"term": "legacy", "tags": []}
]
```

## `poeditor api terms delete`

Delete a term, found by its text and `--context`, or many, with **their translations**. This
cannot be undone, so it needs `--yes`: without it nothing is deleted and the command says what
would go and prints the line to run.

```sh
poeditor api terms delete save --project web --context button --yes
poeditor api terms delete -p 12345 --from-file obsolete.json --yes
```

| Flag | |
|---|---|
| `--context <context>` | The term's context. |
| `--from-file <file>` | Delete the terms listed in a JSON file instead: `[{"term": "save", "context": "button"}, {"term": "old_key"}]`. |
| `--yes` | Delete. |
| `--json` | Print the counts (parsed, deleted) as JSON. |

To make a project's terms exactly a given list, deleting all the others, see
[`poeditor api projects sync`](projects.md#poeditor-api-projects-sync).

## `poeditor api terms add-comment`

Comment on a term, found by its text and `--context`, or on many. The comment belongs to the
term, so translators of every language see it.

```sh
poeditor api terms add-comment save "Max 10 characters" --project web --context button
poeditor api terms add-comment welcome "Keep it short" -p 12345 --pinned
poeditor api terms add-comment -p web --from-file comments.json
```

| Flag | |
|---|---|
| `--context <context>` | The term's context. |
| `--pinned` | Pin the comment. |
| `--from-file <file>` | Add the comments of a JSON file instead. |
| `--json` | Print the counts (parsed, commented) as JSON. |

```json
[
  {"term": "save", "context": "button", "comment": "Max 10 characters", "pinned": 1},
  {"term": "welcome", "comment": "Shown once, on first launch"}
]
```
