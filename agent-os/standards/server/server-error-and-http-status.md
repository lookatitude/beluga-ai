# ServerError and HTTP Status

**ServerError** (iface): Op, Err, Code (string), Message, Details (optional). Implements `Error()` and `Unwrap() error`. **HTTPStatus() int** maps Code to HTTP status; used by REST when writing error responses.

**Error codes** (iface.ErrorCode, snake_case): HTTP—invalid_request, method_not_allowed, not_found, internal_error, timeout, rate_limited, unauthorized, forbidden; MCP—tool_not_found, resource_not_found, tool_execution_error, resource_read_error, invalid_tool_input, mcp_protocol_error; server—server_startup_error, server_shutdown_error, config_validation_error.

**Constructors:** `NewInvalidRequestError`, `NewNotFoundError`, `NewInternalError`, `NewTimeoutError`, `NewToolNotFoundError`, `NewResourceNotFoundError`, `NewToolExecutionError`, `NewResourceReadError`, `NewInvalidToolInputError`, `NewConfigValidationError`, `NewMCPProtocolError`. `server/errors.go` re-exports and adds **IsServerError**, **AsServerError**.

**REST JSON error shape:** `{"error": {"code": "<Code>", "message": "<Message>", "details": <optional>, "operation": "<Op>"}}`. Always set Content-Type application/json and HTTP status from `HTTPStatus()`.
