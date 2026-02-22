module github.com/Hafuunano/Plugin-Collections

go 1.25.6

require (
	github.com/Hafuunano/Protocol-ConvertTool v0.0.0-20260219210233-d010567d8319
	github.com/glebarez/sqlite v1.11.0
	gopkg.in/yaml.v3 v3.0.1
	gorm.io/gorm v1.25.12
)

// Local development: use sibling UniTransfer (module name is Protocol-ConvertTool).
replace github.com/Hafuunano/Protocol-ConvertTool => ../UniTransfer

require (
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/glebarez/go-sqlite v1.21.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/ncruces/go-strftime v0.1.9 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	golang.org/x/sys v0.22.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	modernc.org/libc v1.55.3 // indirect
	modernc.org/mathutil v1.6.0 // indirect
	modernc.org/memory v1.8.0 // indirect
	modernc.org/sqlite v1.23.1 // indirect
)
