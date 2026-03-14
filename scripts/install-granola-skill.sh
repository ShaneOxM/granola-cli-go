#!/usr/bin/env bash

set -euo pipefail

RAW_URL="https://raw.githubusercontent.com/ShaneOxM/granola-cli-go/main/docs/skills/Granola/SKILL.md"

tmp_file="$(mktemp)"
cleanup() {
  rm -f "$tmp_file"
}
trap cleanup EXIT

if command -v curl >/dev/null 2>&1; then
  curl -fsSL "$RAW_URL" -o "$tmp_file"
elif command -v wget >/dev/null 2>&1; then
  wget -qO "$tmp_file" "$RAW_URL"
else
  echo "Error: curl or wget is required to download the Granola skill." >&2
  exit 1
fi

targets=(
  "$HOME/.config/opencode:$HOME/.config/opencode/skills/Granola/SKILL.md"
  "$HOME/.codex:$HOME/.codex/skills/Granola/SKILL.md"
  "$HOME/.claude:$HOME/.claude/skills/Granola/SKILL.md"
  "$HOME/.factory:$HOME/.factory/skills/Granola/SKILL.md"
)

for entry in "${targets[@]}"; do
  base_dir="${entry%%:*}"
  target="${entry#*:}"

  if [[ ! -d "$base_dir" ]]; then
    echo "Skipping $target (base directory not found: $base_dir)"
    continue
  fi

  mkdir -p "$(dirname "$target")"
  cp "$tmp_file" "$target"
  echo "Installed $target"
done

echo "Granola skill installed from GitHub: $RAW_URL"
