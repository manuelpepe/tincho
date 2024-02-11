package front

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
)

//go:embed static/*
var static embed.FS

func FrontendHandler() (http.Handler, error) {
	fsub, err := fs.Sub(static, "static")
	if err != nil {
		return nil, fmt.Errorf("error subbing fs: %w", err)
	}
	return http.FileServer(http.FS(fsub)), nil
}
