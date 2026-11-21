#!/usr/bin/env sh
#
# The single definition of what this project's tasks mean.
#
# The task runner (make / just / npm), the git hooks and CI all shell out to
# this file, so there is exactly one place where "lint" is defined and the
# three entry points cannot drift apart.
#
# Usage: scripts/tasks.sh <task> [args...]
#        scripts/tasks.sh help

set -eu

say() { printf '\033[1m==>\033[0m %s\n' "$*"; }
die() { printf 'error: %s\n' "$*" >&2; exit 1; }
have() { command -v "$1" >/dev/null 2>&1; }

# git-cliff may come from PATH (mise) or node_modules (npm stack).
cliff() {
	if have git-cliff; then
		git-cliff "$@"
	elif [ -x node_modules/.bin/git-cliff ]; then
		node_modules/.bin/git-cliff "$@"
	else
		die "git-cliff not found. Run: scripts/tasks.sh setup"
	fi
}

# lefthook, likewise.
run_lefthook() {
	if have lefthook; then
		lefthook "$@"
	elif [ -x node_modules/.bin/lefthook ]; then
		node_modules/.bin/lefthook "$@"
	else
		die "lefthook not found. Run: scripts/tasks.sh setup"
	fi
}

# Flavor: go library

# Tools mise does not have on PATH fall back to `go run pkg@version`, so a
# contributor without mise can still run every task.
GOVULNCHECK_VERSION="${GOVULNCHECK_VERSION:-v1.10.0}"
PKGSITE_VERSION="${PKGSITE_VERSION:-v0.7.0}"
DOCS_ADDR="${DOCS_ADDR:-localhost:6060}"

run_govulncheck() {
	if have govulncheck; then
		govulncheck "$@"
	else
		go run "golang.org/x/vuln/cmd/govulncheck@$GOVULNCHECK_VERSION" "$@"
	fi
}

run_pkgsite() {
	if have pkgsite; then
		pkgsite "$@"
	else
		go run "golang.org/x/pkgsite/cmd/pkgsite@$PKGSITE_VERSION" "$@"
	fi
}

flavor_setup() {
	say "Downloading Go modules"
	go mod download
}

task_fmt() {
	say "Formatting Go sources"
	if [ "$#" -eq 0 ]; then
		gofmt -w .
		return 0
	fi
	# The hook hands us every staged file; keep only the ones gofmt understands.
	# Append the matches to the positional parameters and drop the originals, so
	# paths containing spaces survive.
	staged_count=$#
	for f in "$@"; do
		case "$f" in *.go) set -- "$@" "$f" ;; esac
	done
	shift "$staged_count"
	[ "$#" -gt 0 ] || return 0
	gofmt -w -- "$@"
}

task_lint() {
	say "Vetting"
	go vet ./...
	unformatted="$(gofmt -l .)"
	[ -z "$unformatted" ] || die "unformatted files:
$unformatted"
	if have golangci-lint; then
		say "golangci-lint"
		golangci-lint run
	else
		say "golangci-lint not installed; skipping (run: scripts/tasks.sh setup)"
	fi
	say "Checking go.mod is tidy"
	# -diff reports what tidying would change and exits non-zero, rather than
	# rewriting go.mod during a task that is supposed to change nothing.
	go mod tidy -diff || die "go.mod/go.sum are not tidy; run 'go mod tidy'"
}

task_vuln() {
	say "Scanning for known vulnerabilities (govulncheck)"
	run_govulncheck ./...
}

task_test() {
	say "Running tests"
	go test -race -shuffle=on -covermode=atomic -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | tail -n 1
}

task_cover() {
	task_test
	say "Writing coverage.html"
	go tool cover -html=coverage.out -o coverage.html
}

task_bench() {
	say "Running benchmarks"
	go test -run='^$' -bench=. -benchmem ./...
}

task_fuzz() {
	fuzztime="${1:-30s}"
	say "Fuzzing every Fuzz* target for $fuzztime each"
	# go test -fuzz accepts exactly one target per package per run.
	for pkg in $(go list ./...); do
		for target in $(go test -list '^Fuzz' "$pkg" | grep '^Fuzz' || true); do
			say "$pkg $target"
			go test -run='^$' -fuzz="^${target}\$" -fuzztime="$fuzztime" "$pkg"
		done
	done
}

# A library has no binary: build just proves every package (and every OS the
# CI matrix covers) compiles.
task_build() {
	say "Compiling all packages"
	go build ./...
}

task_docs() {
	say "Serving pkg.go.dev-style docs at http://$DOCS_ADDR (Ctrl-C to stop)"
	run_pkgsite -open=false -http="$DOCS_ADDR" .
}

task_clean() {
	say "Removing build artifacts"
	rm -rf dist coverage.out coverage.html
}

# Go has no version-bearing manifest: the git tag is the version. Nothing to
# bump, so nothing to stage.
task_bump_version() { :; }

# Everything below is shared by every flavor.

# Bootstrap reacts to which files the template has, so the mise-based and
# npm-based stacks share one code path.
task_setup() {
	if [ -f mise.toml ]; then
		have mise || die "mise not found. Install it: https://mise.jdx.dev"
		say "Installing pinned tools"
		mise install
	fi
	if [ -f package.json ]; then
		say "Installing npm dependencies"
		if [ -f package-lock.json ]; then npm ci; else npm install; fi
	fi
	flavor_setup
	task_hooks
	say "Ready."
}

# Everything release.sh runs before it will cut a tag.
task_preflight() {
	task_lint
	task_vuln
	task_test
}

task_changelog() {
	say "Regenerating CHANGELOG.md"
	cliff --config cliff.toml --output CHANGELOG.md
}

task_release() {
	[ "$#" -ge 1 ] || die "usage: scripts/tasks.sh release <vX.Y.Z|major|minor|patch>"
	./release.sh "$@"
}

task_hooks() {
	# setup runs before `git init` often enough to be worth handling: lefthook
	# exits non-zero with no git directory, which would fail the whole install.
	# .git is a file rather than a directory in worktrees and submodules.
	if [ ! -e .git ]; then
		say "Not a git repository yet; skipping hooks. After 'git init', run: scripts/tasks.sh hooks"
		return 0
	fi
	say "Installing git hooks"
	run_lefthook install
}

task_help() {
	cat <<'USAGE'
Tasks:
  setup        install the toolchain, dependencies and git hooks
  fmt          format sources in place
  lint         static checks; fails on a problem, changes nothing
  vuln         scan dependencies and code with govulncheck
  test         run the tests with -race and write coverage.out
  cover        test, then write coverage.html
  bench        run the benchmarks
  fuzz [time]  run every fuzz target (default 30s each)
  build        compile every package
  docs         preview the pkg.go.dev page locally with pkgsite
  changelog    regenerate CHANGELOG.md from the commit history
  release      cut a release (vX.Y.Z | major | minor | patch)
  preflight    lint + vuln + test, as run before a release
  hooks        (re)install the git hooks
  clean        remove build artifacts
  help         this list

Through this project's task runner:
  make test
  make release V=v1.2.3
USAGE
}

task="${1:-help}"
[ "$#" -gt 0 ] && shift

case "$task" in
	setup) task_setup "$@" ;;
	fmt) task_fmt "$@" ;;
	lint) task_lint "$@" ;;
	vuln) task_vuln "$@" ;;
	test) task_test "$@" ;;
	cover) task_cover "$@" ;;
	bench) task_bench "$@" ;;
	fuzz) task_fuzz "$@" ;;
	build) task_build "$@" ;;
	docs) task_docs "$@" ;;
	changelog) task_changelog "$@" ;;
	release) task_release "$@" ;;
	preflight) task_preflight "$@" ;;
	bump-version) task_bump_version "$@" ;;
	hooks) task_hooks "$@" ;;
	clean) task_clean "$@" ;;
	help | -h | --help) task_help ;;
	*) die "unknown task '$task'. Try: scripts/tasks.sh help" ;;
esac
