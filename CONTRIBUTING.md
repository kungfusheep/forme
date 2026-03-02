# Contributing to glyph

glyph is in active development. Contributions are welcome.

## What's useful right now

- Bug reports — especially unexpected behaviour, panics, or terminal state left in a bad state
- API feedback — anything that felt awkward or required more code than expected
- Real-world usage — if you build something with glyph, sharing what broke or frustrated you is valuable
- Documentation fixes — incorrect or missing examples in the API reference

## Before opening a PR

- Check existing issues to avoid duplicate work
- For non-trivial changes, open an issue first to discuss the approach
- Keep changes focused — one thing per PR

## Running tests

```bash
go test ./...
```

Benchmarks:

```bash
go test -bench=. -benchmem ./...
```

## Code style

Standard Go. `gofmt` formatted. No external dependencies beyond what's already in `go.mod`.

## Reporting issues

Use [GitHub Issues](https://github.com/kungfusheep/glyph/issues). Include:

- Go version and OS
- Terminal emulator
- Minimal reproduction if possible
