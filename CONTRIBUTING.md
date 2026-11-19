# Contributing

Thanks for helping with the POEditor CLI. Bug reports, fixes and new features are all welcome.

## Before you start

- **Bugs:** open an issue with the CLI version (`poeditor --version`), your OS, the command you
  ran and what it printed.
- **Features:** open an issue first so we can agree on the command and flag names before you
  write the code.
- **Security problems:** don't open an issue, see [SECURITY.md](SECURITY.md).

## Building

You need the Go version named in [`go.mod`](go.mod).

```sh
go build ./cmd/poeditor
./poeditor --help
```

The CLI talks only to the public [POEditor API v2](https://poeditor.com/docs/api).

## Tests are required

Every command and every flag has a test, and the suite fails when one is missing:

- Command tests run the real root command against a fake API (`internal/testutil`) and check
  the request sent, the output and the error.
- Expected output lives in `testdata/*.golden` files. After changing output, regenerate them
  with `UPDATE_GOLDEN=1 go test ./...` and read every changed file before committing.
- A bug fix comes with a test that fails without the fix.
- A new command or flag is also documented in `README.md` or the matching `docs/*.md` file;
  the docs tests fail otherwise.

Before opening a pull request, all of these must pass:

```sh
gofmt -l .      # prints nothing
go vet ./...
go test ./...
```

CI runs the same checks on Linux, macOS and Windows.

## Pull requests

- One change per pull request; keep it focused.
- Explain what changed and why in the description, and link the issue it closes.
- By submitting a pull request you agree that your contribution is licensed under the
  [MIT License](LICENSE).
