#!/usr/bin/env bash

set -euo pipefail

RAW_BASE="https://raw.githubusercontent.com/ShaneOxM/granola-cli-go/main/docs/skills/Granola"

tmp_dir="$(mktemp -d)"
cleanup() {
  rm -rf "$tmp_dir"
}
trap cleanup EXIT

download() {
  local url="$1"
  local out="$2"
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$url" -o "$out"
  elif command -v wget >/dev/null 2>&1; then
    wget -qO "$out" "$url"
  else
    echo "Error: curl or wget is required to download the Granola skill." >&2
    exit 1
  fi
}

download "$RAW_BASE/SKILL.md" "$tmp_dir/SKILL.md"

mkdir -p "$tmp_dir/Workflows"
workflow_files=(
  "MeetingList.md"
  "MeetingView.md"
  "MeetingTranscript.md"
  "MeetingNotes.md"
)
for wf in "${workflow_files[@]}"; do
  download "$RAW_BASE/Workflows/$wf" "$tmp_dir/Workflows/$wf"
done

targets=(
  "$HOME/.config/opencode:$HOME/.config/opencode/skills/Granola"
  "$HOME/.codex:$HOME/.codex/skills/Granola"
  "$HOME/.claude:$HOME/.claude/skills/Granola"
  "$HOME/.factory:$HOME/.factory/skills/Granola"
)

for entry in "${targets[@]}"; do
  base_dir="${entry%%:*}"
  target_dir="${entry#*:}"

  if [[ ! -d "$base_dir" ]]; then
    echo "Skipping $target_dir (base directory not found: $base_dir)"
    continue
  fi

  mkdir -p "$target_dir"
  rm -rf "$target_dir/Workflows"
  cp "$tmp_dir/SKILL.md" "$target_dir/SKILL.md"
  cp -R "$tmp_dir/Workflows" "$target_dir/Workflows"
  echo "Installed $target_dir"
done

echo "Granola skill installed from GitHub: $RAW_BASE"
