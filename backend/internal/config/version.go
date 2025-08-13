package config

var (
	// Version 程序版本号，由编译时注入
	Version = "dev"
	// BuildTime 编译时间，由编译时注入
	BuildTime = "unknown"
	// GitCommit Git提交哈希，由编译时注入
	GitCommit = "unknown"
)
