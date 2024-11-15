#!/bin/bash

grep '^	' go.mod | while read line; do
	pkg="$(awk '{print $1}' <<< "$line")"
	version="$(awk '{print $2}' <<< "$line")"
	echo "go get $pkg@$version"

	if grep -q "$pkg" api/go.mod; then
	  ( cd api && go get "$pkg"@"$version" && go mod tidy )
	  make tidy-all
	fi

	if grep -q "$pkg" sdk/go.mod; then
	  ( cd sdk && go get "$pkg"@"$version" && go mod tidy )
	  make tidy-all
	fi
done
