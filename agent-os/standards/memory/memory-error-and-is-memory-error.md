# MemoryError and IsMemoryError

**MemoryError** uses Op, Err, Code, and **Message** (aligned with the framework op-err-code pattern). **Code** is always one of the ErrCode* constants; providers may set a custom **Message**.

- **MemoryError:** Op, Err, Code, Message. Prefer Message for extra context; no Context map — standardize with other packages.
- **ErrCode* constants:** ErrCodeInvalidConfig, ErrCodeInvalidInput, ErrCodeStorageError, ErrCodeRetrievalError, ErrCodeTimeout, ErrCodeNotFound, ErrCodeTypeMismatch, ErrCodeSerialization, ErrCodeDeserialization, ErrCodeValidation, ErrCodeMemoryOverflow, ErrCodeContextCanceled. Use when creating and when checking. **Code must always be one of these** (or package-defined ErrCode*); no arbitrary code strings.
- **Provider-specific messages:** Providers use the same ErrCode* and may set a custom Message (e.g. NewMemoryErrorWithMessage(op, ErrCodeStorageError, "redis connection refused", err)).
- **IsMemoryError(err, code):** code must be an ErrCode* constant. Use for branching and retries.
- **Registry:** Use ErrCodeTypeMismatch when the requested memory type is not registered. Typed helpers: ErrInvalidConfig, ErrStorageError, ErrRetrievalError, etc., in errors.go.
