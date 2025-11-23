package patterns

// getFilePath extracts file path from test function.
func getFilePath(function *TestFunction) string {
	if function.File != nil {
		return function.File.Path
	}
	return ""
}
