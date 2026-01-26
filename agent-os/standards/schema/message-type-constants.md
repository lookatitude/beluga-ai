# MessageType Constants

**Minimum set:** `RoleHuman`, `RoleAssistant` (`"ai"`), `RoleSystem`, `RoleTool`, `RoleFunction`. More roles are allowed when needed.

**Re-export:** `schema` re-exports `MessageType` and the role constants from `iface` so callers can use `schema.RoleHuman`, etc. Prefer importing `iface` directly unless that would introduce circular dependencies.
