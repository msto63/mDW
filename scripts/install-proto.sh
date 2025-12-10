#!/bin/bash
# Script to install protobuf compiler and Go plugins

set -e

echo "Installing protobuf tools..."

# Check OS
OS="$(uname -s)"
ARCH="$(uname -m)"

# Install protoc if not present
if ! command -v protoc &> /dev/null; then
    echo "Installing protoc..."

    PROTOC_VERSION="25.1"

    case "$OS" in
        Darwin)
            if [ "$ARCH" = "arm64" ]; then
                PROTOC_ZIP="protoc-${PROTOC_VERSION}-osx-aarch_64.zip"
            else
                PROTOC_ZIP="protoc-${PROTOC_VERSION}-osx-x86_64.zip"
            fi
            ;;
        Linux)
            if [ "$ARCH" = "aarch64" ]; then
                PROTOC_ZIP="protoc-${PROTOC_VERSION}-linux-aarch_64.zip"
            else
                PROTOC_ZIP="protoc-${PROTOC_VERSION}-linux-x86_64.zip"
            fi
            ;;
        *)
            echo "Unsupported OS: $OS"
            exit 1
            ;;
    esac

    curl -LO "https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/${PROTOC_ZIP}"
    unzip -o "$PROTOC_ZIP" -d "$HOME/.local"
    rm "$PROTOC_ZIP"

    echo "protoc installed to ~/.local/bin"
    echo "Add ~/.local/bin to your PATH if not already present"
else
    echo "protoc is already installed: $(protoc --version)"
fi

# Install Go plugins
echo "Installing Go protobuf plugins..."
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

echo ""
echo "Installation complete!"
echo ""
echo "Verify installation:"
echo "  protoc --version"
echo "  which protoc-gen-go"
echo "  which protoc-gen-go-grpc"
