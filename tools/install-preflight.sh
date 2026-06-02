#!/usr/bin/env bash
# Extract the bundled Preflight plugin tree into the project's .claude/ tree.
# The marketplace manifest (.claude/plugins/.claude-plugin/marketplace.json) and
# the registration in .claude/settings.json are committed to the repo, so this
# installer only unpacks the plugin itself. It is a project-local install and
# does not touch ~/.claude/.

set -euo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
ARCHIVE="${REPO_ROOT}/tools/preflight.tar.gz"
PLUGIN_DIR="${REPO_ROOT}/.claude/plugins/preflight"

if [[ ! -f "$ARCHIVE" ]]; then
  echo "preflight install: archive not found at $ARCHIVE" >&2
  echo "Are you running this from the workshop repo root?" >&2
  exit 1
fi

if ! command -v jq >/dev/null 2>&1; then
  echo "preflight install: jq is required by the plugin's stamp and check scripts." >&2
  echo "Install it with: brew install jq   (macOS)   or   apt-get install jq   (Linux)" >&2
  exit 1
fi

if [[ -d "$PLUGIN_DIR" ]]; then
  echo "preflight install: $PLUGIN_DIR already exists. Re-running overwrites the extracted plugin files."
  read -r -p "Continue? [y/N] " reply
  if [[ ! "$reply" =~ ^[Yy]$ ]]; then
    echo "Aborted."
    exit 0
  fi
  rm -rf "$PLUGIN_DIR"
fi

echo "Extracting Preflight plugin into $PLUGIN_DIR ..."
mkdir -p "$PLUGIN_DIR"
tar -xzf "$ARCHIVE" -C "$PLUGIN_DIR"

# Make sure all bundled shell scripts are executable on this checkout.
find "$PLUGIN_DIR/scripts" "$PLUGIN_DIR/hooks" -type f -name '*.sh' -exec chmod +x {} +
[[ -f "$PLUGIN_DIR/git-hooks/pre-push" ]] && chmod +x "$PLUGIN_DIR/git-hooks/pre-push"

echo
echo "Preflight installed."
echo
echo "What landed:"
echo "  .claude/plugins/preflight/   plugin tree (manifest, agents, hooks, scripts)"
echo
echo "The marketplace manifest and the preflight@preflight-local registration in"
echo ".claude/settings.json are committed to the repo, so they are already in place."
echo
echo "Next steps:"
echo "  1. Restart Claude Code so it picks up the plugin."
echo "  2. Run  /preflight:preflight  to gate your branch (or  /preflight  if there's no naming collision)."
echo "  3. Run  /preflight:install-preflight-hook  to drop the native git pre-push hook."
echo
echo "To uninstall: delete .claude/plugins/preflight/. To unregister entirely, also"
echo "  remove the preflight-local / preflight@preflight-local keys from"
echo "  .claude/settings.json and delete .claude/plugins/.claude-plugin/marketplace.json."
