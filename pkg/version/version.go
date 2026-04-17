package version

var (
	// Version 版本号，通过 -ldflags 注入
	Version = "dev"
	// GitCommit Git 提交哈希，通过 -ldflags 注入
	GitCommit = "unknown"
	// BuildTime 构建时间，通过 -ldflags 注入
	BuildTime = "unknown"
)
