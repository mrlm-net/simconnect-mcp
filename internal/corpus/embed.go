package corpus

import "embed"

//go:generate go run ../../tools/scraper/main.go -out ./assets/
//go:embed assets/*.json
var assetsFS embed.FS
