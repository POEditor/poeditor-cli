# `.poeditor.yml`

`.poeditor.yml` tells `poeditor pull` and `poeditor push` which POEditor projects your
repository uses, where each language file lives, and how to sync it. Commit it: it holds no
tokens.

The file is found like git finds `.git`: in the current directory, then in each parent
directory. Paths inside it are relative to the file's own directory and always use `/`, on
Windows too.

[← back to the README](../README.md)

## An example

```yaml
projects:
  - id: 12345
    alias: web                     # optional nickname: --project web
    account: client-a              # optional; omitted → the default account
    source_language: en            # the language push uploads terms from
    files:
      - path: locales/{lang}.json
        format: key_value_json
      - path: ios/{lang}.lproj/InfoPlist.strings
        format: apple_strings
        languages: [en, fr, de]    # optional; default: every project language
        language_map:              # optional; POEditor code → code in the path
          pt-br: pt-BR
        pull:                      # how the file is exported
          tags: [infoplist]
          tags_match: all
          filters: [translated]
          order: terms
          fallback_language: en
        push:                      # how the file is uploaded
          updating: terms_translations
          overwrite: false
          sync_terms: false
          read_from_source: false
          fuzzy_trigger: false
          tags:
            all: [infoplist]
            new: [needs-review]
            obsolete: [removed]
            overwritten_translations: [changed]
```

You rarely write it by hand: [`poeditor config init`](#poeditor-config-init) builds it from
your project and the files it finds.

## Reference

### Projects

`projects` is a list, even with one project.

| Key | |
|---|---|
| `id` | The POEditor project id (required). `poeditor api projects list` shows them. |
| `alias` | A short local name, so you can write `--project web` instead of the id. Not all digits. |
| `account` | The [account](auth.md#several-accounts) whose token this project uses. Omitted: the default account. |
| `source_language` | The language your code is written in. `push` uploads this language's file to create terms. |
| `files` | The project's files (a list): one project often spans several files, split by tags. |

### Files

| Key | |
|---|---|
| `path` | Where the file lives, with `{lang}` for the language: `locales/{lang}.json`, `res/values-{lang}/strings.xml`. `{lang}` may be left out only when `languages` has a single entry. |
| `format` | The POEditor file format, by its API name: `key_value_json`, `json`, `po`, `pot`, `mo`, `android_strings`, `apple_strings`, `xliff`, `xliff_1_2`, `xlf`, `stringsdict`, `xcstrings`, `arb`, `i18next`, `yml`, `properties`, `resx`, `resw`, `ini`, `csv`, `xls`, `xlsx`, `ts`, `xmb`, `xtb`, `rise_360_xliff`. `mo` can be pulled, not pushed. |
| `languages` | Only these languages. Default: every language of the project. |
| `language_map` | When the path spells a language differently from POEditor: `pt-br: pt-BR` (iOS), `pt-br: pt-rBR` (Android). |
| `pull` | Export options, below. |
| `push` | Upload options, below. |

### `pull:` options

These are the [API's export parameters](https://poeditor.com/docs/api#projects_export), under
their API names.

| Key | |
|---|---|
| `tags` | Only the terms with these tags. |
| `tags_match` | How `tags` match: `all` (every tag), `any` (at least one, the default) or `exclude` (none of them). |
| `filters` | Only the terms that are: `translated`, `untranslated`, `fuzzy`, `not_fuzzy`, `automatic`, `not_automatic`, `ai`, `not_ai`, `tm`, `not_tm`, `proofread`, `not_proofread`. Several must all match. |
| `order` | `terms`: sort alphabetically. |
| `fallback_language` | Fill untranslated terms from this language. |

### `push:` options

These are the [API's upload parameters](https://poeditor.com/docs/api#projects_upload).

| Key | |
|---|---|
| `updating` | What the source file brings: `terms`, `terms_translations` (default) or `translations`. Translation files are always uploaded as `translations`. |
| `overwrite` | Replace existing translations. |
| `sync_terms` | **Deletes** the project's terms that are not in the source file. |
| `read_from_source` | Read translations from the source text (xliff). |
| `fuzzy_trigger` | Mark the other languages' translations fuzzy where the source changed. |
| `tags` | Tags to put on terms: `all` (every term of the file), `new`, `obsolete` (terms missing from the file), `overwritten_translations`. Never copied from `pull.tags`: write them yourself. |

With `sync_terms: true`, `push.tags.obsolete` does nothing: POEditor deletes obsolete terms
instead of tagging them. `poeditor config show` warns about it.

### Validation

The file is checked strictly, so a typo fails loudly instead of syncing nothing: unknown keys,
unknown filter names (the API would silently drop them), bad `tags_match` or `updating` values,
`push` on a pull-only format, two projects with the same alias, two languages written to the
same path. An unknown `format` is only a warning, in case POEditor has added one.

## `poeditor config init`

Create `.poeditor.yml` in the current directory with a short wizard:

1. the account (asked only when you have several);
2. the project, from your project list;
3. a short alias for it, and its source language (default: the project's reference language);
4. its files. The wizard looks for files in this directory whose path holds a language code
   of the project, in the spellings platforms use (`pt-br`, `pt-BR`, `pt_BR`, Android's
   `pt-rBR`), and offers them as `{lang}` patterns, with `language_map` filled in when the
   spelling differs. It guesses the format from the extension, then asks for the tags to pull
   and the tags to put on pushed terms.

`init` refuses to overwrite an existing file, and needs an interactive terminal. Nothing is
written if you cancel with Ctrl+C.

## `poeditor config add`

Add a project, or a file of an existing project, with the same wizard. Only the new entry is
written: your comments and everything else in the file stay as they are.

## `poeditor config show`

Show what the file resolves to: each project, its account, and each file's path per language.
Nothing is sent to POEditor. Problems in the file are listed on stderr; errors make the command
fail. `--json` prints the projects as JSON.

## `poeditor config check`

Check the file against POEditor: for each project, that its account has a token, that the
token can open the project, and that every language the file names (`source_language`,
`languages`, `language_map`, `pull.fallback_language`) exists in the project. Tags can't be
checked: the API has no list of them.

It exits with an error when anything is wrong, so it fits at the start of a CI job.
`--json` prints the report as JSON.

```sh
poeditor config check && poeditor pull
```
