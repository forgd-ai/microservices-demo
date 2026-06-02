---
description: Install the bundled Preflight plugin into this project's .claude/ tree.
---

Run the bundled installer to extract the Preflight plugin from `tools/preflight.tar.gz`
into `.claude/plugins/preflight/`.

The marketplace manifest (`.claude/plugins/.claude-plugin/marketplace.json`) and the
registration in `.claude/settings.json` (`extraKnownMarketplaces.preflight-local` plus
`enabledPlugins."preflight@preflight-local"`) are committed to the repo, so they are
already present on checkout. The extracted plugin tree is gitignored and local, so it
survives `git clean -ffd` and is the only thing the installer creates.

Steps:

1. Run from the repo root:

   ```
   bash tools/install-preflight.sh
   ```

   The script:
   - extracts the plugin tree into `.claude/plugins/preflight/`
   - leaves `~/.claude/` untouched (project-local install)

2. Surface the script's stdout to the user verbatim -- it lists what landed where
   and how to invoke the plugin's commands.

3. Tell the user to **restart Claude Code** so the plugin loads. Without a
   restart, the new slash commands and hooks will not be available in the
   current session.

4. After restart the user can run:
   - `/preflight:preflight` -- gate the current branch (the `/preflight:` prefix
     comes from the plugin name; if there's no naming collision Claude may
     also accept `/preflight`)
   - `/preflight:install-preflight-hook` -- drop the native git pre-push hook
   - `/preflight:preflight-pr` -- generate a PR description from the last
     preflight report

If `tools/preflight.tar.gz` is missing or `jq` is not installed, the script
prints an actionable error and exits non-zero -- relay it to the user as-is.

To remove Preflight afterwards: delete `.claude/plugins/preflight/`. To unregister
it entirely, also remove the `preflight-local` and `preflight@preflight-local` keys
from `.claude/settings.json` and delete `.claude/plugins/.claude-plugin/marketplace.json`.
