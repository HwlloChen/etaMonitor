package static

import (
	"embed"
	"io/fs"
	"net/http"
)

// GetAssetsFileSystem 返回嵌入式 assets 目录
func GetAssetsFileSystem() http.FileSystem {
	assets, err := fs.Sub(distFS, "frontend-dist/assets")
	if err != nil {
		panic(err)
	}
	return http.FS(assets)
}

//go:embed frontend-dist/*
var distFS embed.FS

// GetFileSystem 返回嵌入式文件系统
func GetFileSystem() http.FileSystem {
	dist, err := fs.Sub(distFS, "frontend-dist")
	if err != nil {
		panic(err)
	}
	return http.FS(dist)
}
