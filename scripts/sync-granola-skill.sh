#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "$0")/.." && pwd)"
source_dir="$repo_root/docs/skills/Granola"
source_skill="$source_dir/SKILL.md"

if [[ ! -f "$source_skill" ]]; then
  echo "Granola skill source not found: $source_skill" >&2
  exit 1
fi

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
  cp "$source_skill" "$target_dir/SKILL.md"
  if [[ -d "$source_dir/Workflows" ]]; then
    cp -R "$source_dir/Workflows" "$target_dir/Workflows"
  fi
  echo "Synced $target_dir"
done

echo "Granola skill synced from repo source: $source_dir"
