package main

import (
	"go/build"
	"os"
	"path/filepath"

	"github.com/tkerber/golem/webkit"
)

func webkitInit() {
	// TODO figure out a better way to reference this. (i.e. without the source)
	extenPath := ""
	for _, src := range build.Default.SrcDirs() {
		p := filepath.Join(src, "github.com", "tkerber", "golem", "web_extension")
		if _, err := os.Stat(p); err == nil {
			extenPath = p
			break
		}
	}
	if extenPath == "" {
		panic("Failed to find source files!")
	}

	c := webkit.GetDefaultWebContext()
	c.SetWebExtensionsDirectory(extenPath)
	// TODO this is temporary.
	c.RegisterURIScheme("golem", golemSchemeHandler)
	// NOTE: removing this will cause bugs in golems web extension.
	// Tread lightly.
	c.SetProcessModel(webkit.ProcessModelMultipleSecondaryProcesses)
}

func golemSchemeHandler(req *webkit.URISchemeRequest) {
	req.Finish([]byte("<html><head><title>Golem</title></head><body><h1>Golem Home Page</h1><p>And stuff.</p></body></html>"), "text/html")
}
