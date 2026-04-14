#!/bin/bash
# UserPromptSubmit hook: Scan user prompt for secrets before submission.
# Blocks the prompt if secrets are detected.

if ! command -v sonar &> /dev/null; then
  exit 0
fi

stdin_data=$(cat)
prompt=$(echo "$stdin_data" | sed -n 's/.*"prompt"[[:space:]]*:[[:space:]]*"\(.*\)"[[:space:]]*}.*/\1/p' | head -1)

if [[ -z "$prompt" ]]; then
  exit 0
fi

tmpfile=$(mktemp)
trap 'rm -f "$tmpfile"' EXIT
printf '%s' "$prompt" > "$tmpfile"

sonar analyze secrets "$tmpfile" > /dev/null 2>&1
exit_code=$?

if [[ $exit_code -eq 51 ]]; then
  reason="Sonar detected secrets in user prompt"
  echo "{\"hookSpecificOutput\":{\"hookEventName\":\"UserPromptSubmit\",\"permissionDecision\":\"deny\",\"permissionDecisionReason\":\"$reason\"}}"
  exit 0
fi

exit 0
