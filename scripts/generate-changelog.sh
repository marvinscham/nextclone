#!/usr/bin/env bash
set -euo pipefail

output="CHANGELOG.md"
section_only="false"
next_version=""

while [ "$#" -gt 0 ]; do
	case "$1" in
		--output)
			output="${2:?output path is required}"
			shift 2
			;;
		--section-only)
			section_only="true"
			shift
			;;
		--version)
			next_version="${2:?version is required}"
			shift 2
			;;
		*)
			echo "unknown argument: $1" >&2
			exit 1
			;;
	esac
done

repo_url="${GITHUB_SERVER_URL:-https://github.com}/${GITHUB_REPOSITORY:-}"
if [ -z "${GITHUB_REPOSITORY:-}" ]; then
	origin="$(git remote get-url origin 2>/dev/null || true)"
	case "$origin" in
		git@github.com:*) repo_url="https://github.com/${origin#git@github.com:}" ;;
		https://github.com/*) repo_url="${origin%.git}" ;;
	esac
	repo_url="${repo_url%.git}"
fi

commit_url() {
	printf '%s/commit/%s' "$repo_url" "$1"
}

category_label() {
	case "$1" in
		feat) printf 'Feature' ;;
		fix) printf 'Fix' ;;
		perf) printf 'Performance' ;;
		refactor) printf 'Refactor' ;;
		docs) printf 'Documentation' ;;
		test) printf 'Test' ;;
		build) printf 'Build' ;;
		ci) printf 'CI' ;;
		chore) printf 'Chore' ;;
		revert) printf 'Revert' ;;
		style) printf 'Style' ;;
		*) return 1 ;;
	esac
}

commit_entry() {
	local sha="$1"
	local subject="$2"
	local type=""
	local title=""
	local conventional_regex='^([a-zA-Z]+)(\([^)]+\))?!?:[[:space:]]*(.+)$'

	if [[ "$subject" =~ $conventional_regex ]]; then
		type="${BASH_REMATCH[1],,}"
		title="${BASH_REMATCH[3]}"
	else
		return 0
	fi

	if [ "$type" = "chore" ] && [[ "$subject" =~ ^chore\(release\): ]]; then
		return 0
	fi

	if ! category_label "$type" >/dev/null; then
		return 0
	fi

	printf '%s\t- [%s](%s)\n' "$(category_label "$type")" "$title" "$(commit_url "$sha")"
}

write_section() {
	local version="$1"
	local range="$2"
	local tmp=""
	tmp="$(mktemp)"

	while IFS=$'\t' read -r sha subject; do
		commit_entry "$sha" "$subject" >> "$tmp"
	done < <(git log --no-merges --reverse --format='%H%x09%s' "$range")

	if [ ! -s "$tmp" ]; then
		rm -f "$tmp"
		return 0
	fi

	printf '## %s\n\n' "$version"
	for category in Feature Fix Performance Refactor Documentation Test Build CI Chore Revert Style; do
		if grep -q "^${category}"$'\t' "$tmp"; then
			printf '### %s\n\n' "$category"
			grep "^${category}"$'\t' "$tmp" | cut -f2-
			printf '\n'
		fi
	done

	rm -f "$tmp"
}

target="$(mktemp)"
if [ "$section_only" != "true" ]; then
	printf '# Changelog\n\n' > "$target"
fi

mapfile -t tags < <(git tag --list 'v[0-9]*' --sort=v:refname)

if [ -n "$next_version" ]; then
	if [[ "$next_version" != v* ]]; then
		next_version="v${next_version}"
	fi
	next_tag_index=""
	for ((i=0; i<${#tags[@]}; i++)); do
		if [ "${tags[$i]}" = "$next_version" ]; then
			next_tag_index="$i"
			break
		fi
	done
	if [ -n "$next_tag_index" ]; then
		if [ "$next_tag_index" -gt 0 ]; then
			write_section "$next_version" "${tags[$((next_tag_index-1))]}..${next_version}" >> "$target"
		else
			write_section "$next_version" "$next_version" >> "$target"
		fi
	else
		latest_tag=""
		if [ "${#tags[@]}" -gt 0 ]; then
			latest_tag="${tags[-1]}"
		fi
		if [ -n "$latest_tag" ]; then
			write_section "$next_version" "${latest_tag}..HEAD" >> "$target"
		else
			write_section "$next_version" "HEAD" >> "$target"
		fi
	fi
fi

for ((i=${#tags[@]}-1; i>=0; i--)); do
	tag="${tags[$i]}"
	if [ "$tag" = "$next_version" ]; then
		continue
	fi
	if [ "$section_only" = "true" ] && [ -n "$next_version" ] && [ "$tag" != "$next_version" ]; then
		continue
	fi
	if [ "$i" -gt 0 ]; then
		write_section "$tag" "${tags[$((i-1))]}..${tag}" >> "$target"
	else
		write_section "$tag" "$tag" >> "$target"
	fi
done

mv "$target" "$output"
