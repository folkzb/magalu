#!/bin/bash

if [ -z "$VERSION" ]; then
    VERSION=`git log -1 '--pretty=format:%(describe:tags)'`
fi

FILE="mgc/sdk/version.txt"
VERSION_TXT=$(cat "$FILE")
if [ "$VERSION" != "$VERSION_TXT" ]; then
    echo "Versions differ: have=$VERSION_TXT, want=$VERSION. Please update $FILE"
    exit 1
fi

echo "Versions match: $VERSION"
