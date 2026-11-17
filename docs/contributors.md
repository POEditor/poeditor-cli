# Contributors

`poeditor api contributors` shows who has access to your projects, gives people access to a
language or a whole project, and takes it away.

There are two kinds of access:

- an **administrator** manages the whole project;
- a **contributor** translates one or more languages, and can also be a **proofreader** of them
  when the project has proofreading on.

[← back to the README](../README.md)

## `poeditor api contributors list`

Who has access: one line per person and project, with their role (`administrator`,
`proofreader` or `contributor`) and languages.

```console
$ poeditor api contributors list
ID     PROJECT     NAME     EMAIL            ROLE           LANGUAGES
12345  Web         Ana Pop  ana@example.com  proofreader    fr, de
67890  Mobile app  Ana Pop  ana@example.com  contributor    fr
12345  Web         Bo       bo@example.com   administrator
```

| Flag | |
|---|---|
| `--project <alias or id>` (`-p`) | Only this project. Without it: every project of the account. |
| `--language <code>` (`-l`) | Only the contributors of this language. Needs `--project`. |
| `--json` | Print the contributors as JSON, each with their permissions per project. |

## `poeditor api contributors add`

Give someone access to a project: one language to translate, optionally as proofreader, or the
whole project as administrator. POEditor invites people who don't have an account yet.

```sh
poeditor api contributors add ana@example.com --name Ana --project web --language fr
poeditor api contributors add ana@example.com --name Ana -p 12345 -l fr --proofreader
poeditor api contributors add bo@example.com --name Bo -p web --admin
```

| Flag | |
|---|---|
| `--project <alias or id>` (`-p`) | The project (required). |
| `--name <name>` | The person's name (required). |
| `--language <code>` (`-l`) | The language to translate. One of `--language` or `--admin` is required. |
| `--proofreader` | A proofreader of the language too. The project needs proofreading on. Someone who already translates the language becomes a proofreader. |
| `--admin` | An administrator of the project. Only the project's owner can add one. |

## `poeditor api contributors remove`

Take someone's access away: all of it, one language, or only the proofreader right. It needs
`--yes`: without it nothing is removed and the command prints the line to run.

```sh
poeditor api contributors remove ana@example.com --project web --yes
poeditor api contributors remove ana@example.com -p 12345 --language fr --yes
poeditor api contributors remove ana@example.com -p web --proofreader --yes
```

| Flag | |
|---|---|
| `--project <alias or id>` (`-p`) | The project (required). |
| `--language <code>` (`-l`) | Only from this language. |
| `--proofreader` | Only the proofreader right; they keep translating. |
| `--yes` | Remove. |

Only the project's owner can remove an administrator.
