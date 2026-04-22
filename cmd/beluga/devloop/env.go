package devloop

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// envKeyAllowed reports whether the given environment-variable key is
// acceptable in a `.env` file. The grammar is the one documented by POSIX
// for identifiers — leading letter/underscore, subsequent letters,
// digits, or underscores — so no shell metacharacters can appear in a
// key and accidental leading whitespace is rejected cleanly.
func envKeyAllowed(key string) bool {
	if key == "" {
		return false
	}
	for i, r := range key {
		switch {
		case r == '_':
		case r >= 'A' && r <= 'Z':
		case r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9':
			if i == 0 {
				return false
			}
		default:
			return false
		}
	}
	return true
}

// LoadProjectEnv reads a `.env` file from the scaffolded project root
// and returns its contents as a slice of "KEY=value" entries in file
// order, suitable for concatenation with os.Environ().
//
// The path is resolved via filepath.EvalSymlinks and the result must be
// contained inside projectRoot (after EvalSymlinks of projectRoot
// itself), to prevent a symlinked `.env` from reading an arbitrary file
// outside the project tree.
//
// Missing `.env` is not an error — LoadProjectEnv returns (nil, nil)
// so `beluga run` on a project without a `.env` is a valid flow.
//
// Syntax: `KEY=value` per line, blank lines and `#` comments permitted.
// Values may be quoted with single or double quotes to preserve leading
// whitespace; inside double quotes, `\n`, `\r`, `\t`, and `\"` escape
// sequences are expanded. Any line that is not blank, a comment, or a
// valid assignment is a parse error.
func LoadProjectEnv(projectRoot string) ([]string, error) {
	rootReal, err := filepath.EvalSymlinks(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve project root: %w", err)
	}

	envPath := filepath.Join(rootReal, ".env")
	envReal, err := filepath.EvalSymlinks(envPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("resolve .env: %w", err)
	}

	rel, err := filepath.Rel(rootReal, envReal)
	if err != nil || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return nil, fmt.Errorf(".env escapes project root: %s", envReal)
	}

	// #nosec G304 -- envReal is bounded to projectRoot above; this
	// path is supplied by the developer running `beluga run` in their
	// own project tree, not by a remote caller.
	f, err := os.Open(envReal)
	if err != nil {
		return nil, fmt.Errorf("open .env: %w", err)
	}
	defer func() { _ = f.Close() }()

	var out []string
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	lineno := 0
	for sc.Scan() {
		lineno++
		line := sc.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		eq := strings.IndexByte(trimmed, '=')
		if eq <= 0 {
			return nil, fmt.Errorf(".env:%d: expected KEY=value", lineno)
		}
		key := trimmed[:eq]
		if !envKeyAllowed(key) {
			return nil, fmt.Errorf(".env:%d: invalid key %q", lineno, key)
		}
		val, err := parseEnvValue(trimmed[eq+1:])
		if err != nil {
			return nil, fmt.Errorf(".env:%d: %w", lineno, err)
		}
		out = append(out, key+"="+val)
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("read .env: %w", err)
	}
	return out, nil
}

// parseEnvValue strips surrounding quotes and expands escapes for a
// double-quoted form. Unquoted values are returned verbatim with
// trailing comments (` # ...`) stripped so `KEY=value # comment` behaves
// as developers expect.
func parseEnvValue(raw string) (string, error) {
	if raw == "" {
		return "", nil
	}
	switch raw[0] {
	case '\'':
		return parseSingleQuoted(raw)
	case '"':
		return parseDoubleQuoted(raw)
	}
	return stripInlineComment(raw), nil
}

func parseSingleQuoted(raw string) (string, error) {
	end := strings.IndexByte(raw[1:], '\'')
	if end < 0 {
		return "", fmt.Errorf("unterminated single-quoted value")
	}
	rest := strings.TrimSpace(raw[1+end+1:])
	if rest != "" && !strings.HasPrefix(rest, "#") {
		return "", fmt.Errorf("unexpected text after closing quote: %q", rest)
	}
	return raw[1 : 1+end], nil
}

func parseDoubleQuoted(raw string) (string, error) {
	var b strings.Builder
	i := 1
	for ; i < len(raw); i++ {
		c := raw[i]
		if c == '"' {
			rest := strings.TrimSpace(raw[i+1:])
			if rest != "" && !strings.HasPrefix(rest, "#") {
				return "", fmt.Errorf("unexpected text after closing quote: %q", rest)
			}
			return b.String(), nil
		}
		if c == '\\' && i+1 < len(raw) {
			i++
			switch raw[i] {
			case 'n':
				b.WriteByte('\n')
			case 'r':
				b.WriteByte('\r')
			case 't':
				b.WriteByte('\t')
			case '"':
				b.WriteByte('"')
			case '\\':
				b.WriteByte('\\')
			default:
				b.WriteByte('\\')
				b.WriteByte(raw[i])
			}
			continue
		}
		b.WriteByte(c)
	}
	return "", fmt.Errorf("unterminated double-quoted value")
}

func stripInlineComment(raw string) string {
	if idx := strings.Index(raw, " #"); idx >= 0 {
		raw = raw[:idx]
	}
	return strings.TrimRight(raw, " \t")
}

// MergeEnv layers base (typically os.Environ()), loaded (from
// LoadProjectEnv), and extras (Config.ExtraEnv) so that later sources
// override earlier ones for duplicate keys. The returned slice is safe
// to pass to exec.Cmd.Env.
func MergeEnv(base, loaded, extras []string) []string {
	// Clamp the cap hint via int64 + a conservative upper bound so the
	// sum can never overflow a native int (CodeQL go/allocation-size-
	// overflow). 1<<20 env entries is already five orders of magnitude
	// beyond any real system; `append` will grow the slice past that
	// clamp on the unreachable-in-practice hot path.
	const mergeEnvCapLimit = 1 << 20
	capHint := int64(len(base)) + int64(len(loaded)) + int64(len(extras))
	if capHint > mergeEnvCapLimit {
		capHint = mergeEnvCapLimit
	}
	order := make([]string, 0, int(capHint))
	idx := make(map[string]int)
	add := func(entry string) {
		key, _, ok := strings.Cut(entry, "=")
		if !ok {
			return
		}
		if pos, seen := idx[key]; seen {
			order[pos] = entry
			return
		}
		idx[key] = len(order)
		order = append(order, entry)
	}
	for _, e := range base {
		add(e)
	}
	for _, e := range loaded {
		add(e)
	}
	for _, e := range extras {
		add(e)
	}
	return order
}
