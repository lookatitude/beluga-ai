# Function and Type Godoc

Every exported function and type MUST have a godoc comment. First sentence is the summary; add Parameters, Returns, and Example when they help. Use structured blocks so docs and code are easier to understand when scanning documentation or code.

- **First sentence:** One-line summary; start with the name or a verb (e.g. "NewX creates ...", "GetMetrics returns ..."). Required.
- **Parameters:** When it helps:
  - `Parameters:`
  - `  - name: short description`
  Use for non-obvious or multiple parameters.
- **Returns:** When it helps:
  - `Returns:`
  - `  - Type: description` (or `  - error: when ...`)
  Use for multiple return values or when error conditions matter.
- **Example:**
  - **Short usage (small footprint):** In-comment `Example:` with a few indented lines. Use when the snippet stays brief.
  - **Longer usage:** Use `Example usage can be found in examples/.../main.go` (or the real path). Use when in-comment would be too long or need more context.
- **Error conditions:** For functions that return errors, mention when and why they fail (in Returns or a short line).
