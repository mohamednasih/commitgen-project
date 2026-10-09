![Example](example.png)

# CommitGen

CommitGen is a self-contained Go CLI that examines staged Git changes and uses
Google Gemini to suggest a Conventional Commit title and optional description.

## Features

- Generates Conventional Commit titles (`feat`, `fix`, `chore`, and others).
- Creates optional bullet-point descriptions.
- Supports accepting, regenerating, or editing generated text.
- Installs as one native executable with no runtime dependencies.

## Requirements

- Git
- Go 1.22 or newer, or Docker (only needed to build from source)
- A Gemini API key

## Install from source

```sh
git clone https://github.com/N0ViP/commitgen-project.git
cd commitgen-project
./install.sh
```

The installer builds `commitgen` into `$GOBIN`, or `~/.local/bin` when `GOBIN`
is unset. If Go is unavailable, it automatically builds with the official Go
Docker image. Make sure the installation directory is included in your `PATH`.

You can also build without installing:

```sh
go build -o commitgen ./cmd/commitgen
```

## Configuration

Create a Gemini API key in [Google AI Studio](https://ai.google.dev/) and expose
it to CommitGen:

```sh
export GEMINI_API_KEY="your_api_key_here"
```

The model can be overridden when necessary:

```sh
export COMMITGEN_MODEL="gemini-3.8-flash"
```

To keep these values between sessions, add the exports to your shell profile,
such as `~/.bashrc` or `~/.zshrc`.

## Usage

Stage changes and start CommitGen from anywhere inside the repository:

```sh
git add .
commitgen
```

Follow the prompts to accept, regenerate, edit, or skip generated content. The
default editor is `nano` on Unix-like systems and `notepad` on Windows. Set
`EDITOR` to override it.

CommitGen sends the staged diff to the Gemini API. Do not stage secrets or other
sensitive information you do not want sent to Google.

## Development

```sh
go test ./...
go vet ./...
```

## Uninstall

Remove the installed executable:

```sh
rm ~/.local/bin/commitgen
```

If you set `GOBIN` during installation, remove `commitgen` from that directory
instead.
