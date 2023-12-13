#!/bin/bash

# Step 1: Detect OS and Architecture
OS=$(uname | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

# For ARM architecture, handle the ARM version
if [[ $ARCH == arm* ]]; then
    ARM_VERSION=$(echo $ARCH | sed 's/armv//')
    ARCH="arm"
else
    ARM_VERSION=""
fi

# Step 2: Define GitHub Repository and Binary Name
REPO="edgeflare/edge"
BINARY_NAME="edge"

# Step 3: Handle Version Selection
VERSION=${1:-latest}

# Fetch the latest release tag from GitHub if 'latest' is selected
if [ "$VERSION" == "latest" ]; then
    VERSION=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/."([^"]+)"./\1/')
fi

# Step 4: Compose Download URL
FILENAME="${BINARY_NAME}v${VERSION}${OS}_${ARCH}${ARM_VERSION:+v$ARM_VERSION}"
EXTENSION="tar.gz"
[ "$OS" == "windows" ] && EXTENSION="zip"

DOWNLOAD_URL="https://github.com/$REPO/releases/download/$VERSION/$FILENAME.$EXTENSION"

# Step 5: Download and Extract Archive
curl -L $DOWNLOAD_URL -o "$FILENAME.$EXTENSION"
mkdir -p $BINARY_NAME
tar -xzf "$FILENAME.$EXTENSION" -C $BINARY_NAME --strip-components=1 || unzip "$FILENAME.$EXTENSION" -d $BINARY_NAME

# Step 6: Make Binary Executable
chmod +x "$BINARY_NAME/$BINARY_NAME"

# Step 7: Move Binary to a directory in PATH (e.g., /usr/local/bin)
# Uncomment the line below if you want to move the binary instead of executing it directly
# mv "$BINARY_NAME/$BINARY_NAME" /usr/local/bin/

# Optionally, run the binary
# "./$BINARY_NAME/$BINARY_NAME"

# Clean up
rm -rf $BINARY_NAME
rm "$FILENAME.$EXTENSION"

echo "Installation complete."

# End of script