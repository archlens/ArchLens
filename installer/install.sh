#!/usr/bin/env bash

# Work in a temp directory so nothing lands in (or collides with) the current directory
tmp=$(mktemp -d)

__cleanup ()
{
    rm -rf "$tmp"
}

trap __cleanup EXIT

OS=$(uname -s)
ARCH=$(uname -m)

# Map the OS and architecture to the release asset names
case "$OS" in
    Linux)  os="linux" ;;
    Darwin) os="darwin" ;;
    *)
        echo "Unsupported OS: $OS (Windows users: download the zip from the releases page)"
        exit 1
        ;;
esac

case "$ARCH" in
    x86_64|amd64)  arch="amd64" ;;
    arm64|aarch64) arch="arm64" ;;
    *)
        echo "Unsupported architecture: $ARCH"
        exit 1
        ;;
esac

URL="https://github.com/archlens/ArchLensGo/releases/latest/download/archlens-${os}-${arch}.tar.gz"

# Download and extract
echo "Downloading $URL"
if ! curl -fsSL -o "$tmp/archlens.tar.gz" "$URL"; then
    echo "Failed to download $URL"
    exit 1
fi

echo "Untar gzip"
if ! tar -xzf "$tmp/archlens.tar.gz" -C "$tmp"; then
    echo "Failed to extract the archive"
    exit 1
fi

echo "Finding binary file"
# The binary may sit at the archive root or in a folder, so locate it
bin=$(find "$tmp" -type f -name archlens | head -n1)
if [ -z "$bin" ]; then
    echo "Could not find the archlens binary in the archive"
    exit 1
fi

echo "chmod bin"
chmod +x "$bin"
if ! "$bin" --help >/dev/null 2>&1; then
    echo "The downloaded binary failed to run"
    exit 1
fi

# Choose where to install it
target="/usr/local/bin/archlens"
input=""
if { : </dev/tty; } 2>/dev/null; then
    printf 'Enter the path to install the binary to (default: %s): ' "$target"
    read -r input </dev/tty
fi
target=${input:-$target}

# A directory means "put archlens in there"
if [ -d "$target" ]; then
    target="${target%/}/archlens"
fi

if [ ! -d "$(dirname "$target")" ]; then
    echo "Directory does not exist: $(dirname "$target")"
    exit 1
fi

# Only ask for sudo when the destination isn't writable
if [ -w "$(dirname "$target")" ]; then
    install -m 0755 "$bin" "$target"
else
    sudo install -m 0755 "$bin" "$target"
fi
if [ $? -ne 0 ]; then
    echo "Failed to move the binary to $target"
    exit 1
fi

echo "Binary moved to $target successfully"
