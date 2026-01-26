---
description: Run Agent OS project installation for this repo (project-install.sh with profile beluga)
---

# Agent OS Project Install

## User Input

```text
$ARGUMENTS
```

You **MUST** consider the user input before proceeding (if not empty). Use it to override the default profile or pass flags (e.g. `--commands-only`, `--verbose`).

## Goal

Install Agent OS into this project so you can use standards, discover-standards, inject-standards, and sync-to-profile. This creates `agent-os/` and `agent-os/standards/`, and installs Agent OS commands.

## Steps

1. **Ensure project root**: Run from the repository root (e.g. where `go.mod` or project config lives). If unsure, `cd` to the workspace root first.

2. **Choose profile and flags**:
   - Default profile: **beluga**
   - If the user provided a name in `$ARGUMENTS`, use it as `--profile <name>`
   - If the user wrote `--commands-only` or `--verbose`, forward those flags

3. **Run the installer**:

   ```bash
   ~/agent-os/scripts/project-install.sh --profile beluga
   ```

   Replace `beluga` with the chosen profile if different. Add `--commands-only` to only update commands and preserve existing `agent-os/standards/`. Add `--verbose` for detailed output.

4. **Handle prompts**: If the script asks to overwrite existing `agent-os/standards/`, the user must answer in the terminal. Do not assume yes/no; tell the user to respond.

5. **Report and next steps**: After it finishes, report:
   - What was created/updated (`agent-os/`, `agent-os/standards/`, `index.yml`, commands in `.claude/commands/agent-os/`)
   - Next steps:
     - Run `/discover-standards` (or the Agent OS discover-standards command) to extract patterns into standards
     - Run inject-standards when building features
   - For Cursor: if they want Agent OS commands in Cursor, copy or symlink `.claude/commands/agent-os/` to `.cursor/commands/agent-os/`.

## Troubleshooting

- **`No standards directory`** when running `sync-to-profile.sh`: run this install first.
- **`Cannot install in the base installation directory`**: they are in `~/agent-os`; `cd` to this project and run again.
- **`Profile not found`**: the given profile is missing under `~/agent-os/profiles/`; use an existing profile (e.g. `default`, `beluga`) or create it.
