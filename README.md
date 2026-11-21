# goLibTemplates

[![Go Reference](https://pkg.go.dev/badge/github.com/Smiduweorc/goLibTemplates.svg)](https://pkg.go.dev/github.com/Smiduweorc/goLibTemplates)
[![CI](https://github.com/Smiduweorc/goLibTemplates/actions/workflows/ci.yml/badge.svg?branch=master)](https://github.com/Smiduweorc/goLibTemplates/actions/workflows/ci.yml)

A template for a Go library, built on the [black-garden-ants][bga] `go-make`
template: `go vet`, golangci-lint, govulncheck, race-enabled tests with
coverage, fuzzing, benchmarks, runnable examples for
[pkg.go.dev](https://pkg.go.dev), commit linting, git-cliff and lefthook.
CI runs on Linux, macOS and Windows.

## Starting from it

1. Click **Use this template**, clone, then run `make setup`. That installs
   the toolchain pinned in `mise.toml`, the modules and the git hooks.
2. Change the module path in `go.mod` and the import in `example_test.go`,
   rename the package in every `.go` file, and update the badges above.
3. Replace `greet.go`, `greet_test.go` and `example_test.go` with your code.
   Write the package overview in `doc.go`.

Keep the default branch named `master`: CI, `release.sh` and the docs
workflow all expect it.

## Tasks

`make help` lists them; every one is defined in `scripts/tasks.sh`, which the
hooks and CI call too.

| Task | What it does |
| --- | --- |
| `make lint` | `go vet`, `gofmt`, golangci-lint (`.golangci.yml`), `go mod tidy -diff` |
| `make vuln` | `govulncheck ./...` |
| `make test` | `go test -race -shuffle=on` with `coverage.out` |
| `make cover` | as `test`, plus `coverage.html` |
| `make bench` | benchmarks with allocations |
| `make fuzz T=1m` | each `Fuzz*` target for `T` (default 30s) |
| `make docs` | preview the pkg.go.dev page locally with pkgsite on :6060 |
| `make release V=patch` | cut a release (`vX.Y.Z` \| `major` \| `minor` \| `patch`) |

## Documentation

pkg.go.dev renders the package from the module proxy; nothing is hosted here.
Doc comments on every exported identifier and `Example*` functions in
`example_test.go` are what appears there. On a pull request the Docs
workflow checks the page renders; on a `v*` tag it asks proxy.golang.org for
the new version so pkg.go.dev indexes it straight away. Push the tag with
`PUSH=1 make release V=…` or `git push origin <tag>`.

## Releases

Commit messages must follow [Conventional Commits][cc]. They are checked by
[conventionalcommit/commitlint][ccl], a Go port of commitlint pinned as a
`tool` directive in `go.mod` and configured in `.commitlint.yaml`. The
commit-msg hook runs it, and CI runs it over every commit in a pull request. `make release` runs lint, govulncheck and the tests,
regenerates `CHANGELOG.md`, commits and tags. The tag is the version: Go has
no manifest to bump. For `v2` and later, the module path must end in `/v2`
([Go modules: v2 and beyond](https://go.dev/blog/v2-go-modules)).

[bga]: https://github.com/Smiduweorc/black-garden-ants#readme
[cc]: https://www.conventionalcommits.org
[ccl]: https://github.com/conventionalcommit/commitlint
