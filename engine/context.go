package engine

import "io/fs"

// Context encapsulates the execution context for template rendering
type Context struct {
	// TmplFS is the filesystem containing template files
	TmplFS fs.FS
	// OutputRoot is the root directory for generated files
	OutputRoot string
	// PackagePath is the Go package path for the generated code (future use)
	// This will be used for import resolution and package declarations
	PackagePath string
}

func NewContext(tmplFS fs.FS, outputRoot, packagePath string) Context {
	return Context{
		TmplFS:      tmplFS,
		OutputRoot:  outputRoot,
		PackagePath: packagePath,
	}
}
