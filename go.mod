module github.com/eldada/metrics-viewer

go 1.14

require (
	github.com/andybalholm/brotli v1.0.3 // indirect
	github.com/buger/goterm v1.0.3
	github.com/gdamore/tcell/v2 v2.4.1-0.20210905002822-f057f0a857a1
	github.com/hpcloud/tail v1.0.1-0.20180514194441-a1dbeea552b7
	github.com/jfrog/jfrog-cli-core/v2 v2.3.0
	github.com/jfrog/jfrog-client-go v1.4.0
	github.com/prometheus/client_model v0.2.0
	github.com/prometheus/common v0.30.0
	github.com/rivo/tview v0.0.0-20210920163636-bb872b4b26a0
	github.com/stretchr/testify v1.7.0
	gopkg.in/fsnotify/fsnotify.v1 v1.4.7 // indirect
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7 // indirect
)

replace github.com/jfrog/jfrog-cli-core/v2 => github.com/jfrog/jfrog-cli-core/v2 v2.3.0
