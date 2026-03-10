# Contributing to glyph

glyph is in active development. Contributions are welcome and appreciated; this document provides guidelines to help you get started.

Contributions don't have to be code! Here are some ways you can help:

- Bug reports: unexpected behaviour, panics, or terminal state left in a bad state
- API feedback: anything that felt awkward or needed more code than expected
- Real-world usage: if you built something with glyph, what broke or frustrated you
- Documentation fixes: incorrect or missing examples in the API reference

## Before opening a PR

- Check existing issues to avoid duplicate work
- For non-trivial changes, open an issue first to agree on the approach and avoid duplicate work
- One thing per PR

## Running tests

```bash
go test ./...
```

It's recommended you make sure any code you change is covered by tests - taking into account any contextual nuance that may or may not be represented, and that all tests pass before opening a PR.

Benchmarks:

```bash
go test -bench=. -benchmem ./...
```

Given the performance focus of glyph, it's recommended you run benchmarks before and after your change to ensure it doesn't cause a significant regression. If it does, consider ways to mitigate the regression or open an issue to discuss.

## Code style

Standard Go.
`gofmt` formatted.
No new external dependencies without discussion. 
We do use dot imports when declaring glyph views for readability, but otherwise avoid them.
If in doubt, follow the style of existing code.

## Reporting issues

Use [GitHub Issues](https://github.com/kungfusheep/glyph/issues). Include Go version, OS, terminal emulator, and a minimal reproduction.

You can run `go run ./cmd/dr` to get an issue-friendly diagnostic report. 

    
