# Dual Output Logger

Logger writes to stdout and file with potentially different formats.

```go
func (l *StructuredLogger) log(ctx context.Context, level LogLevel, message string, fields map[string]any) {
    entry := l.createLogEntry(ctx, level, message, fields)
    output := l.formatEntry(entry)

    // Stdout: colored when not JSON
    if l.stdout != nil {
        if l.useColors && !l.useJSON {
            output = l.colorize(level, output)
        }
        l.stdout.Println(output)
    }

    // File: plain text when not JSON (historical implementation)
    if l.fileOut != nil {
        if l.useJSON {
            l.fileOut.Println(output)
        } else {
            plainOutput := l.formatPlainText(entry)
            l.fileOut.Println(plainOutput)
        }
    }

    // FATAL always exits
    if level == FATAL {
        os.Exit(1)
    }
}
```

## Behavior Matrix

| Setting | Stdout | File |
|---------|--------|------|
| JSON=true | JSON | JSON |
| JSON=false, Colors=true | Colored text | Plain text |
| JSON=false, Colors=false | Plain text | Plain text |

## FATAL Level
- Hardcoded to call `os.Exit(1)` after logging
- No error return - immediate termination
- Use ERROR for recoverable failures

## Note
File format differing from stdout when not JSON is a historical implementation detail.
