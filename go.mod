module github.com/eldada/metrics-viewer

go 1.14

require (
	github.com/buger/goterm v0.0.0-20200322175922-2f3e71b85129
	github.com/gdamore/tcell/v2 v2.0.1-0.20201109052606-7d87d8188c8d
	github.com/hpcloud/tail v1.0.1-0.20180514194441-a1dbeea552b7
	github.com/jfrog/jfrog-cli-core v0.0.1
	github.com/jfrog/jfrog-client-go v0.16.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_model v0.2.0
	github.com/prometheus/common v0.15.0
	github.com/rivo/tview v0.0.0-20201204190810-5406288b8e4e
	github.com/stretchr/testify v1.4.0
	gopkg.in/fsnotify/fsnotify.v1 v1.4.7 // indirect
)

replace github.com/jfrog/jfrog-cli-core => github.com/jfrog/jfrog-cli-core v1.1.2
