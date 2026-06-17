# Contributing

Thanks for your interest in improving zest.

## Getting started

```sh
git clone https://github.com/wilmarvh/zest.git
cd zest
go build ./...
go vet ./...
go run .
```

You'll need macOS, the Music app with a populated library, and Xcode Command
Line Tools (the library loader is compiled with CGo against the
`iTunesLibrary` framework).

## Guidelines

- Keep changes focused; one logical change per pull request.
- Run `go build ./...` and `go vet ./...` before submitting — both should be clean.
- Match the existing code style; `gofmt` your code.
- Describe what you changed and how you tested it.

## Reporting bugs

Open an issue with your macOS version, Go version, and clear steps to reproduce.

## License

By contributing, you agree that your contributions will be licensed under the
Apache License, Version 2.0, the same license that covers this project.
