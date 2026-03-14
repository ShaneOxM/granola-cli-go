#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "$0")/.." && pwd)"
source_skill="$repo_root/docs/skills/Granola/SKILL.md"

if [[ ! -f "$source_skill" ]]; then
  echo "Granola skill source not found: $source_skill" >&2
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
  cp "$source_skill" "$target"
  echo "Synced $target"
done

echo "Granola skill synced from repo source: $source_skill"
