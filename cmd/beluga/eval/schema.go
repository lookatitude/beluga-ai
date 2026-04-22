package eval

import _ "embed"

//go:embed schema.json
var datasetSchema []byte

// DatasetSchema returns the embedded JSON Schema for beluga eval
// dataset files (draft 2020-12). The CI-integration workflow validates
// eval-report.json against this schema; the hidden `beluga eval schema`
// subcommand prints it to stdout for editor consumption.
func DatasetSchema() []byte {
	out := make([]byte, len(datasetSchema))
	copy(out, datasetSchema)
	return out
}
