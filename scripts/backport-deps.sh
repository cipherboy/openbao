#!/bin/bash

release_dir="$1"
modules=$(find . -name go.mod)

curl -sSL https://vuln.go.dev/index/modules.json > /tmp/vuln-go-dev-modules.json

function clean_version() { echo "$@" | sed 's/^v//g' | sed 's/-.*$//g' | sed 's/+.*$//g'; }

# Helpers from https://gist.github.com/jonlabelle/6691d740f404b9736116c22195a8d706
function version_gt() { test "$(echo "$@" | tr " " "\n" | sort -V | head -n 1)" != "$1"; }
function version_ge() { test "$(echo "$@" | tr " " "\n" | sort -rV | head -n 1)" == "$1"; }

while read -r go_mod; do
	grep '^	' "$go_mod" | while read -r line; do
		pkg="$(awk '{print $1}' <<< "$line")"
		current_version="$(awk '{print $2}' <<< "$line")"
		update_version="$(awk '{print $2}' <<< "$line")"
		release_version="$(grep "^	${pkg} " "$release_dir/$go_mod" | awk '{print $2}')"

	  if [ "$release_version" == "" ]; then
		  continue
	  fi

	  if [ "$update_version" == "$release_version" ]; then
		  continue
	  fi

	  update_version="$(clean_version "$update_version")"
	  release_version="$(clean_version "$release_version")"

	  if version_gt "$release_version" "$update_version"; then
		  if ! [[  "$pkg" == "github.com/openbao/openbao"* ]]; then
			  echo "Dependency $pkg is newer on release branch than main"
		  fi
		  continue
	  fi

	  entry="$(jq '.[] | select(.path == "'"$pkg"'")' < /tmp/vuln-go-dev-modules.json)"
	  vuln_count="$(jq '.vulns | length' <<< "$entry")"

	  if (( vuln_count == 0 )); then
		  continue
	  fi

	  echo "Checking vulnerabilities for $pkg..."

	  update=false

	  if [ "$update_version" == "0.0.0" ]; then
		  update=true
	  fi

	  for vuln_index in $(seq 0 "$(( vuln_count - 1 ))"); do
		  vuln_info="$(jq '.vulns['"$vuln_index"']' <<< "$entry")"
		  vuln_id="$(jq -r '.id' <<< "$vuln_info")"

		  if ! [[ "$vuln_info" == *fixed* ]]; then
			  echo "-> Skipping $vuln_id as it is not fixed"
			  continue
		  fi

		  fixed_mod_version="$(jq -r '.fixed' <<< "$vuln_info")"
		  fixed_version="$(clean_version "$fixed_mod_version")"

		  if version_gt "$fixed_version" "$release_version" && version_ge "$update_version" "$fixed_version"; then
			  update="true"
			  break
		  elif ! version_ge "$update_version" "$fixed_version"; then
			  echo "-> !!! Update $pkg on main to $fixed_mod_version first !!!"
		  fi
	  done

	  if [ "$update" == "true" ]; then
		  echo "-> Updating on release branch to $current_version"
		  (
			  cd "$release_dir" || exit 1
			  dir="$(dirname "$go_mod")"
			  cd "$dir" || exit 1

			  echo "[$PWD] " go get "$pkg@$current_version"
			  go get "$pkg@$current_version"
		  ) || exit 1
	  fi
	done
done <<< "$modules"

(
	cd "$release_dir" || exit 1
	make sync-deps
) || exit 1
