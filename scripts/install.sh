#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "$0")/.." && pwd)"
binary_path="$repo_root/bin/granola"

build_binary() {
  mkdir -p "$repo_root/bin"
  CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o "$binary_path" ./cmd/granola
}

pick_install_dir() {
  if [ -d "/opt/homebrew/bin" ]; then
    printf "/opt/homebrew/bin"
    return
  fi
  if [ -d "/usr/local/bin" ]; then
    printf "/usr/local/bin"
    return
  fi
  mkdir -p "$HOME/.local/bin"
  printf "%s" "$HOME/.local/bin"
}

backup_existing() {
  local target="$1"
  local backup="$2"

  if [ ! -e "$target" ] && [ ! -L "$target" ]; then
    return
  fi

  if [ -e "$backup" ] || [ -L "$backup" ]; then
    rm -f "$backup"
  fi

  mv "$target" "$backup"
  printf 'Backed up existing granola to %s\n' "$backup"
}

main() {
  build_binary

  local install_dir target backup
  install_dir="$(pick_install_dir)"
  target="$install_dir/granola"
  backup="$install_dir/granola-node"

  if [ -e "$target" ] || [ -L "$target" ]; then
    if [ "$(readlink "$target" 2>/dev/null || true)" != "$binary_path" ]; then
      backup_existing "$target" "$backup"
    else
      rm -f "$target"
    fi
  fi

  ln -sf "$binary_path" "$target"

  printf 'Installed granola to %s\n' "$target"
  if [ "$install_dir" = "$HOME/.local/bin" ]; then
    printf 'Add %s to PATH if needed.\n' "$install_dir"
  fi
  printf 'granola now points to %s\n' "$binary_path"
}

main "$@"
