#!/usr/bin/env sh
set -eu

project_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
install_dir="${GOBIN:-${HOME}/.local/bin}"
linker_flags="-s -w"
embed_key=false

case "${1:-}" in
    "") ;;
    --embed-key)
        if [ -z "${GEMINI_API_KEY:-}" ]; then
            echo "Error: --embed-key requires GEMINI_API_KEY to be set." >&2
            exit 1
        fi
        case "$GEMINI_API_KEY" in
            *[[:space:]]*)
                echo "Error: GEMINI_API_KEY cannot contain whitespace when embedded." >&2
                exit 1
                ;;
        esac
        linker_flags="$linker_flags -X github.com/N0ViP/commitgen-project/internal/commitgen.embeddedAPIKey=$GEMINI_API_KEY"
        embed_key=true
        echo "Warning: the Gemini API key will be extractable from the executable." >&2
        ;;
    -h|--help)
        echo "Usage: ./install.sh [--embed-key]"
        echo "  --embed-key  embed the current GEMINI_API_KEY into the executable (insecure)"
        exit 0
        ;;
    *)
        echo "Error: unknown option: $1" >&2
        echo "Usage: ./install.sh [--embed-key]" >&2
        exit 1
        ;;
esac

mkdir -p "$install_dir"
install_dir=$(CDPATH= cd -- "$install_dir" && pwd)
cd "$project_dir"

if command -v go >/dev/null 2>&1; then
    if [ "$embed_key" = true ]; then
        build_cache=$(mktemp -d "${TMPDIR:-/tmp}/commitgen-build-cache.XXXXXX")
        trap 'rm -rf "$build_cache"' EXIT HUP INT TERM
        GOCACHE="$build_cache" go build -trimpath -ldflags="$linker_flags" -o "$install_dir/commitgen" ./cmd/commitgen
    else
        go build -trimpath -ldflags="$linker_flags" -o "$install_dir/commitgen" ./cmd/commitgen
    fi
elif command -v docker >/dev/null 2>&1; then
    case "$(uname -s)" in
        Linux) target_os=linux; binary_name=commitgen ;;
        Darwin) target_os=darwin; binary_name=commitgen ;;
        MINGW*|MSYS*|CYGWIN*) target_os=windows; binary_name=commitgen.exe ;;
        *)
            echo "Error: unsupported operating system: $(uname -s)" >&2
            exit 1
            ;;
    esac

    case "$(uname -m)" in
        x86_64|amd64) target_arch=amd64 ;;
        arm64|aarch64) target_arch=arm64 ;;
        *)
            echo "Error: unsupported CPU architecture: $(uname -m)" >&2
            exit 1
            ;;
    esac

    echo "Go was not found; building CommitGen with Docker..."
    docker run --rm \
        -e CGO_ENABLED=0 \
        -e "GOOS=$target_os" \
        -e "GOARCH=$target_arch" \
        -v "$project_dir:/src" \
        -v "$install_dir:/out" \
        -w /src \
        golang:1.22-alpine \
        go build -trimpath -ldflags="$linker_flags" -o "/out/$binary_name" ./cmd/commitgen
else
    echo "Error: building CommitGen requires Go 1.22+ or Docker." >&2
    echo "Install one of them and run this script again." >&2
    exit 1
fi

echo "CommitGen installed at $install_dir/${binary_name:-commitgen}"
case ":${PATH}:" in
    *":${install_dir}:"*) ;;
    *) echo "Add $install_dir to PATH to run 'commitgen' from anywhere." ;;
esac
