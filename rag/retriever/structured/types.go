package structured

// QueryResult holds the outcome of a structured query execution.
type QueryResult struct {
	// Query is the generated database query (Cypher or SQL).
	Query string

	// Results contains the rows returned by the query, each represented
	// as a map of column name to value.
	Results []map[string]any

	// Error holds any error that occurred during generation or execution.
	Error error
}

// ColumnInfo describes a single column or property in a schema.
type ColumnInfo struct {
	// Name is the column or property name.
	Name string

	// Type is the data type (e.g. "TEXT", "INTEGER", "STRING").
	Type string

	// Description is an optional human-readable description of the column.
	Description string
}

// TableInfo describes a table (SQL) or node label (Cypher).
type TableInfo struct {
	// Name is the table or node label name.
	Name string

	// Columns lists the columns or properties belonging to this table/label.
	Columns []ColumnInfo

	// Description is an optional human-readable description.
	Description string
}

// RelationshipInfo describes a relationship between tables or node labels.
type RelationshipInfo struct {
	// From is the source table or node label.
	From string

	// To is the target table or node label.
	To string

	// Type is the relationship type (e.g. "FRIENDS_WITH" for Cypher,
	// "foreign_key" for SQL).
	Type string

	// Properties lists any properties on the relationship (Cypher) or
	// columns on the join table (SQL).
	Properties []ColumnInfo
}

// SchemaInfo describes the structure of a database, providing context for
// query generation. It supports both relational (SQL) and graph (Cypher)
// schemas.
type SchemaInfo struct {
	// Tables lists all tables (SQL) or node labels (Cypher) in the schema.
	Tables []TableInfo

	// Relationships lists foreign-key or graph relationships.
	Relationships []RelationshipInfo

	// Dialect identifies the query language (e.g. "sql", "cypher").
	Dialect string

	// ExtraContext is free-form text appended to the generation prompt,
	// useful for providing domain-specific hints.
	ExtraContext string
}
