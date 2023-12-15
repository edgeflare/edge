#!/bin/bash

OS=$(uname | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

if [ "$ARCH" == "x86_64" ]; then
    ARCH="amd64"
fi

REPO="edgeflare/edge"
BINARY_NAME="edge"

VERSION=${1:-latest}

# Fetch the latest release tag from GitHub if 'latest' is selected
if [ "$VERSION" == "latest" ]; then
    VERSION=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')
fi

FILENAME="${BINARY_NAME}_${VERSION}_${OS}_${ARCH}${ARM_VERSION:+v$ARM_VERSION}"
EXTENSION="tar.gz"
[ "$OS" == "windows" ] && EXTENSION="zip"

DOWNLOAD_URL="https://github.com/$REPO/releases/download/$VERSION/$FILENAME.$EXTENSION"

curl -fL $DOWNLOAD_URL -o "$FILENAME.$EXTENSION"
tar -xzf "$FILENAME.$EXTENSION" || unzip "$FILENAME.$EXTENSION"
rm $FILENAME.$EXTENSION

echo "Download complete."
echo "run ./edge -h for help and usage."
echo "optionally, move edge to \$PATH e.g., /usr/local/bin by running:"
echo "sudo mv $BINARY_NAME /usr/local/bin/"
echo "optionally, rm LICENSE README.md"
