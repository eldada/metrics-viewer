module github.com/eldada/metrics-viewer

go 1.14

require (
	github.com/buger/goterm v0.0.0-20200322175922-2f3e71b85129
	github.com/jfrog/jfrog-cli-core v0.0.1
	github.com/jfrog/jfrog-client-go v0.16.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_model v0.2.0
	github.com/prometheus/common v0.15.0
	github.com/rivo/tview v0.0.0-20200712113419-c65badfc3d92
	github.com/stretchr/testify v1.4.0
)

replace github.com/jfrog/jfrog-cli-core => github.com/jfrog/jfrog-cli-core v1.1.2
