package main

import (
	"github.com/eldada/metrics-viewer/commands"
	"github.com/eldada/metrics-viewer/visualization"
	"github.com/jfrog/jfrog-cli-core/v2/plugins"
	"github.com/jfrog/jfrog-cli-core/v2/plugins/components"
)

const version = "v0.3.0"

func main() {
	visualization.SetVersion(version)
	plugins.PluginMain(getApp())
}

func getApp() components.App {
	app := components.App{}
	app.Name = "metrics-viewer"
	app.Description = "Easily present Open Metrics data in terminal."
	app.Version = version
	app.Commands = getCommands()
	return app
}

func getCommands() []components.Command {
	return []components.Command{
		commands.GetGraphCommand(),
		commands.GetPrintCommand(),
	}
}
