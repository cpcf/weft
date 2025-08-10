package engine

import "io/fs"

type Context struct {
	TmplFS      fs.FS
	OutputRoot  string
	PackagePath string
}

func NewContext(tmplFS fs.FS, outputRoot, packagePath string) Context {
	return Context{
		TmplFS:      tmplFS,
		OutputRoot:  outputRoot,
		PackagePath: packagePath,
	}
}