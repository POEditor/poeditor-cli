# pull and push

`poeditor pull` and `poeditor push` sync the files listed in [`.poeditor.yml`](config.md) with
POEditor: every project, every file, every language, in one command.

- **push** sends your source-language files up, so new and changed terms reach POEditor.
- **pull** brings every language down into its file, so translations reach your code.

A typical loop: change strings in code, `poeditor push`, translators work in POEditor,
`poeditor pull` before a release.

For a single file without `.poeditor.yml`, use
[`poeditor api projects export`](projects.md#poeditor-api-projects-export) and
[`poeditor api projects upload`](projects.md#poeditor-api-projects-upload): they share the
same code.

[← back to the README](../README.md)

## `poeditor pull`

For every project, file and language in `.poeditor.yml`, export the file from POEditor (with
the file's `pull:` options) and write it to its path, creating directories as needed.

```console
$ poeditor pull
created        locales/fr.json
updated        locales/de.json
unchanged      locales/en.json
3 files: 1 created, 1 updated, 1 unchanged
```

- **POEditor wins**: local changes to those files are overwritten.
- A file whose content would not change is **left untouched**, so `git status` stays clean.
- Every write is **atomic** (a temporary file, then a rename): an interrupted pull never
  leaves half a file.
- Languages are the file's `languages:` list, or every language of the project.
- Files are downloaded one at a time. Pull **stops at the first error**, naming the file;
  files written before it stay written.

| Flag | |
|---|---|
| `--project <alias or id>` | Only this project. |
| `--language <code>` | Only these languages (repeat, or separate with commas). |
| `--dry-run` | Download and compare, but write nothing: each file is `would create`, `would update` or `unchanged`. |
| `--json` | Print one JSON object per file and language (below), as well as the lines on stderr. |

```sh
poeditor pull --project web --language fr,de
poeditor pull --dry-run
```

## `poeditor push`

Upload the **source-language file** of each file entry (the project's `source_language`), with
`updating: terms_translations` unless the file's `push.updating` says otherwise: new and
changed terms, with their source text.

```console
$ poeditor push
uploaded       locales/en.json (en)  terms: 42 parsed, 3 added, 0 deleted · translations: 3 added, 0 updated
1 file uploaded: 3 terms added, 0 deleted
```

With `--translations`, every other language is uploaded too, as `updating: translations`: a
translation file never creates terms, and whether it overwrites existing translations follows
the file's `push.overwrite`. A translation file that doesn't exist locally yet is skipped
(`missing`), not an error.

| Flag | |
|---|---|
| `--project <alias or id>` | Only this project. |
| `--translations` | Also upload every other language. |
| `--language <code>` | With `--translations`: only these languages (repeat, or separate with commas). The source language is uploaded only if listed. |
| `--dry-run` | List what would be uploaded, with the options, and send nothing. |
| `--json` | Print one JSON object per file and language (below). |

```sh
poeditor push --project web
poeditor push --translations --language fr
poeditor push --dry-run --translations
```

**Checked before anything is sent:** every project has a `source_language`, every format can
be uploaded (`mo` can't), every source file exists. If one fails, nothing is uploaded.

**Upload limits.** POEditor accepts one upload every 10 seconds (20 on free plans). Push
uploads one file at a time, waits 10 seconds between uploads, and when POEditor says it is too
early, waits and retries, up to 3 times (after 10, 20, then 30 seconds). Source files go first,
so terms exist before their translations arrive. Push stops at the first error, naming the file.

**Deleting terms.** A file with `push.sync_terms: true` deletes the project's terms that are
not in its source file. That is what you want when the source file is the whole truth; leave it
off when several files feed one project.

## `--json`

Both commands print an array with one object per file and language:

```json
[
  {
    "project": {"id": 12345, "alias": "web"},
    "file": "locales/en.json",
    "language": "en",
    "status": "uploaded",
    "terms": {"parsed": 42, "added": 3, "deleted": 0},
    "translations": {"parsed": 42, "added": 3, "updated": 0}
  }
]
```

`status` is `created`, `updated` or `unchanged` for pull (`would_create`, `would_update` with
`--dry-run`), and `uploaded`, `missing` or `would_upload` for push. If the run stops on an
error, the array still lists what was done before it, and the exit code is 1.
