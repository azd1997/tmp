#!/bin/bash
echo "start building..."
echo $GOROOT
$GOROOT/go.exe build -o ./ecoind.exe ../cmd/ecoind/