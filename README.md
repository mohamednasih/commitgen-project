# CommitGen

CommitGen is a self-contained Go command-line tool that turns staged Git changes
into a Conventional Commit using Google Gemini. By default it generates both a
title and description, displays the result, and creates the commit without
asking for input. An interactive mode is available when review is required.

## Features

- Generates Conventional Commit titles and concise bullet-point descriptions.
- Commits automatically by default, with an optional interactive review mode.
- Samples every staged file instead of sending only the beginning of a large diff.
- Retries temporary Gemini failures with exponential backoff.
- Aborts without committing when generation ultimately fails or returns empty text.
- Keeps title and description prompts editable outside the Go source.
- Builds as a single executable with no Go runtime required on destination machines.
- Publishes cross-platform binaries and SHA-256 checksums from version tags.

## Requirements

At runtime, CommitGen needs:

- Git available in `PATH`.
- Network access to the Gemini API.
- A `GEMINI_API_KEY` environment variable.

Building from source additionally requires Go 1.22 or newer, or Docker. Users
who download a release binary need neither Go nor Docker.

## Installation

### Release binary

Download the appropriate executable and `checksums.txt` from
[GitHub Releases](https://github.com/N0ViP/commitgen-project/releases).

Available targets are:

| Operating system | Architectures |
| --- | --- |
| Linux | amd64, arm64 |
| macOS | amd64, arm64 |
| Windows | amd64 |

Verify the SHA-256 checksum, rename the executable to `commitgen`
(`commitgen.exe` on Windows), and place it in a directory included in `PATH`.

### Build from source

```sh
git clone https://github.com/N0ViP/commitgen-project.git
cd commitgen-project
./install.sh
```

The installer uses a local Go toolchain when available. Otherwise, it builds
through the official `golang:1.22-alpine` Docker image. The resulting executable
is installed into `$GOBIN`, or `~/.local/bin` when `GOBIN` is unset.

Build manually with:

```sh
go build -trimpath -o commitgen ./cmd/commitgen
```

## Configuration

Create a key in [Google AI Studio](https://ai.google.dev/) and export it:

```sh
export GEMINI_API_KEY="your_api_key_here"
```

CommitGen uses the stable `gemini-3.1-flash-lite` model by default. Override it
when needed:

```sh
export COMMITGEN_MODEL="gemini-3.8-flash"
```

Add these exports to a shell profile such as `~/.bashrc` or `~/.zshrc` to keep
them between sessions.

### Embedding the API key

Environment-based configuration is strongly recommended. An API key embedded in
an executable is not secret: anyone who receives the binary can extract and use
it, and rotating the key requires rebuilding every copy.

If a private, single-user binary still needs a built-in key, use the explicit
installer option with a newly created key:

```sh
export GEMINI_API_KEY="your_new_api_key"
./install.sh --embed-key
```

The installer prints a security warning and embeds the key at link time. The
executable then works when `GEMINI_API_KEY` is unset. Setting the environment
variable at runtime overrides the embedded key, allowing emergency rotation.

Never publish, upload, or distribute an executable containing an embedded key.
Official release binaries never contain a key.

## Usage

Stage exactly the changes that belong in the commit, then run CommitGen:

```sh
git add path/to/changed-files
commitgen
```

Automatic mode performs the following operations without application prompts:

1. Reads the staged filenames and diff summary.
2. Builds a bounded sample containing context from every staged file.
3. Generates a Conventional Commit title.
4. Generates a description using that title and the staged context.
5. Displays the completed message and executes `git commit`.

### Interactive mode

Use interactive mode to accept, regenerate, edit, skip, or cancel generated
content before committing:

```sh
commitgen --interactive
```

The short form is:

```sh
commitgen -i
```

The editor defaults to `nano` on Unix-like systems and `notepad` on Windows.
Set `EDITOR` to override it.

When entering optional description notes, finish with `EOF` on its own line:

```text
Focus on the retry behavior.
Mention that errors abort the commit.
EOF
```

The delimiter leaves standard input open for the remaining interactive prompts.

## Custom prompts

Prompt templates are stored in:

- `prompts/title.txt`
- `prompts/description.txt`

Edit them before running `./install.sh`. They are embedded into the executable
at build time, so the installed program remains a single file. Rebuild after
every prompt change.

The following variables are replaced before each Gemini request:

| Variable | Templates | Value |
| --- | --- | --- |
| `{{FILES}}` | Both | Comma-separated staged filenames |
| `{{DIFF}}` | Both | Diff statistics and fairly sampled staged changes |
| `{{TITLE}}` | Description | Generated or accepted commit title |
| `{{NOTES}}` | Description | Interactive notes, or `None` |

## Large commits

Gemini receives at most 8,000 characters of diff context. CommitGen reserves
space for `git diff --stat`, divides the remaining budget across every staged
file, and reallocates unused space from small files to larger ones. Truncated
file sections are marked explicitly.

This approach prevents files near the end of a large commit from being silently
excluded from the prompt.

## Errors and retries

CommitGen retries transient failures up to three times after the initial call,
using delays of 1, 2, and 4 seconds. Retryable failures include:

- HTTP 408 and 425 responses.
- HTTP 429 rate limits.
- HTTP 5xx server errors.
- Network transport errors.

Authentication and invalid-request errors are not retried. If all retries fail,
or Gemini returns no usable title or description, CommitGen exits with status 1
without invoking `git commit`.

## Privacy

CommitGen sends staged filenames, diff statistics, and sampled staged content to
Google Gemini. Unstaged changes are not intentionally included. Review the Git
staging area before running CommitGen, and do not stage credentials, private
keys, customer information, or other material that must not leave your machine.

## Development

Run the test suite and static analysis:

```sh
go test ./...
go vet ./...
```

The project uses only Go's standard library. Tests cover prompt substitution,
Gemini retries, heredoc-style input, response handling, and fair diff sampling.

## Publishing a release

Pushing a tag beginning with `v` runs the release workflow, tests the project,
cross-compiles all supported binaries, generates `checksums.txt`, and creates a
GitHub release:

```sh
git tag v0.2.0
git push origin v0.2.0
```

The workflow is defined in `.github/workflows/release.yml`.

## Uninstall

Remove the installed executable:

```sh
rm ~/.local/bin/commitgen
```

If `GOBIN` was set during installation, remove `commitgen` from that directory
instead.

## License

See [LICENSE.txt](LICENSE.txt).
