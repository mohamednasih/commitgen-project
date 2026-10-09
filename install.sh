#!/usr/bin/env sh
set -eu

project_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
install_dir="${GOBIN:-${HOME}/.local/bin}"
mkdir -p "$install_dir"
install_dir=$(CDPATH= cd -- "$install_dir" && pwd)
cd "$project_dir"

if command -v go >/dev/null 2>&1; then
    go build -trimpath -ldflags="-s -w" -o "$install_dir/commitgen" ./cmd/commitgen
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
        go build -trimpath -ldflags="-s -w" -o "/out/$binary_name" ./cmd/commitgen
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
