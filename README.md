# JFrog CLI metrics-viewer Plugin
A JFrog CLI plugin or standalone binary to show [open-metrics](https://openmetrics.io/) formatted data in a terminal based graph.

- JFrog Artifactory
![The Metrics Viewer Graph - single metric](images/metrics-viewer-graph.png)

- JFrog Artifactory (multiple metrics)
![The Metrics Viewer Graph - multiple metrics](images/metrics-viewer-graph-multiple.png)

## About this plugin
This JFrog CLI plugin is for viewing JFrog products metrics in real time in a terminal. 

## Building from source
To build the **metrics-viewer** binary
```shell
go build
```
To build the **metrics-viewer** binary for multiple operating systems and architectures (Mac, Linux and Windows)
```shell
./build-binary.sh
```


# Testing the code
```shell
# Just run the tests
go test ./...

# Run the tests and create a coverage report
mkdir -p out && go test -coverprofile=out/coverage.out ./... && go tool cover -html=out/coverage.out
```

## Building a Docker image
To build the **metrics-viewer** into a Docker image and use it
```shell
# Build the Docker image
docker build -t metrics-viewer:0.3.0 .

# Test the Docker image
docker run --rm metrics-viewer:0.3.0 --version
```

## Installation of local binary with JFrog CLI
If you don't want to install the plugin from the [JFrog CLI Plugins Registry](https://github.com/jfrog/jfrog-cli-plugins-reg), it needs to be built and installed manually.<br>

Follow these steps to install and use this plugin with JFrog CLI.
1. Make sure JFrog CLI is installed on you machine by running ```jf```. If it is not installed, [install](https://jfrog.com/getcli/) it.
2. Create a directory named ```plugins``` under ```~/.jfrog/``` if it does not exist already.
3. Clone this repository.
4. CD into the root directory of the cloned project.
5. Run ```go build``` to create the binary in the current directory.
6. Copy the binary into the ```~/.jfrog/plugins``` directory.

## Installation with JFrog CLI (when plugin is in the JFrog CLI Plugins Registry)
Installing the latest version:
```shell
jf plugin install metrics-viewer
```

Installing a specific version:
```shell
jf plugin install metrics-viewer@version
```

Uninstalling a plugin
```shell
jf plugin uninstall metrics-viewer
```

## Artifactory Metrics
You can view [Artifactory Metrics](https://jfrog.com/help/r/jfrog-platform-administration-documentation/open-metrics) in various ways.

To try it out, you can run Artifactory as Docker container.

* [Start Artifactory as a Docker](https://jfrog.com/help/r/jfrog-installation-setup-documentation/install-artifactory-single-node-with-docker) container and [enable its metrics](https://jfrog.com/help/r/jfrog-platform-administration-documentation/open-metrics)

* Once Artifactory is up, you can see the metrics log file or [REST API endpoint](https://jfrog.com/help/r/jfrog-rest-apis/get-the-open-metrics-for-artifactory)
```shell
# See the metrics log files. For example, Artifactory and Metadata logs
cat artifactory/log/artifactory-metrics.log
cat artifactory/log/metadata-metrics.log

# Get the metrics from Artifactory REST API
curl -s -uadmin:password http://localhost:8082/artifactory/api/v1/metrics
```

## Usage
### Commands
#### As JFrog CLI plugin
The **metrics-viewer** can be run as a JFrog CLI Plugin or directly as a binary
- **Usage**
```shell
jf metrics-viewer <command> [options]
```  

- **Commands**: See available commands by just running the binary
```shell
jf metrics-viewer
jf metrics-viewer help
```

- **Options**: To see available options for each command, call it with the help
```shell
jf metrics-viewer help graph 
jf metrics-viewer help print 
```
#### As a standalone binary
- **Usage**
```shell
./metrics-viewer <command> [options]
```  

- **Commands**: See available commands by just running the binary
```shell
./metrics-viewer
./metrics-viewer help
```

- **Options**: To see available options for each command, call it with the help
```shell
./metrics-viewer help graph 
./metrics-viewer help print 
```

### Examples as JFrog CLI plugin
- Using the **metrics-viewer** binary
```shell
# Use with the default Artifactory that is configured by the JFrog CLI
jf metrics-viewer graph

# Use with direct Artifactory metrics API URL
jf metrics-viewer graph --url http://localhost:8082/artifactory/api/v1/metrics --user admin --password password

# Use with direct Metadata metrics API URL (NOTE: must get an access token from Artifactory)
jf metrics-viewer graph --url http://localhost:8082/metadata/api/v1/metrics --token ${TOKEN}

# Print metrics of the default Artifactory that is configured by the JFrog CLI
jf metrics-viewer print

# Print metrics of the "art17" Artifactory with name matching the "app_" filter
jf metrics-viewer print --server-id art17 --filter 'app_.*'

# Print selected Artifactory metrics as CSV
jf metrics-viewer print --url http://localhost:8082/artifactory/api/v1/metrics --user admin --password password \
    --format csv --metrics jfrt_runtime_heap_totalmemory_bytes,jfrt_db_connections_active_total
```

### Examples as standalone binary
- Using the **metrics-viewer** binary
```shell
# Use with the default Artifactory that is configured by the JFrog CLI
./metrics-viewer graph

# Use with direct Artifactory metrics API URL
./metrics-viewer graph --url http://localhost:8082/artifactory/api/v1/metrics --user admin --password password

# Use with direct Metadata metrics API URL (NOTE: must get an access token from Artifactory)
./metrics-viewer graph --url http://localhost:8082/metadata/api/v1/metrics --token ${TOKEN}

# Print metrics of the default Artifactory that is configured by the JFrog CLI
./metrics-viewer print

# Print metrics of the "art17" Artifactory with name matching the "app_" filter
./metrics-viewer print --server-id art17 --filter 'app_.*'

# Print selected Artifactory metrics as CSV
./metrics-viewer print --url http://localhost:8082/artifactory/api/v1/metrics --user admin --password password \
    --format csv --metrics jfrt_runtime_heap_totalmemory_bytes,jfrt_db_connections_active_total
```

- Using the Docker image
```shell
# Use with direct Artifactory metrics API URL
# NOTE: The server URL has to be accessible from within the Docker container
docker run --rm --name metrics-viewer metrics-viewer:0.3.0 \
    graph --url http://artifactory-server/artifactory/api/v1/metrics --user admin --password password

# Print specific metrics as CSV
# NOTE: The Docker container needs to access the file system for the logs, so need to mount it into the container 
docker run --rm --name metrics-viewer -v $(pwd)/artifactory:/artifactory metrics-viewer:0.3.0 \
    print --file /artifactory/log/artifactory-metrics.log \
    --format csv --metrics jfrt_runtime_heap_freememory_bytes,jfrt_runtime_heap_totalmemory_bytes
```

### The Viewer
Once running, the viewer will show 3 main sections
- Left pane: Box with selected metrics and another box with list of available metrics (matching search pattern if set)
- Center pane: Graph of selected metrics
- Right pane: Selected metrics metadata and **Max**, **Min** and **Current** values

#### Keys
- Up/Down arrow keys: Move between available metrics
- Space/Enter: Select/Deselect metric to view
- "/": Search pattern in available metrics (supprts regex)
  - Enter to apply pattern and jump back to metrics list
  - ESC to clear search text
- Ctrl+C: Exit **metrics-viewer**

## Release Notes
The release notes are available [here](RELEASE.md).

## Contributions
A big **THANK YOU** to the **real** developers here for joining me for this idea!
- [yinonavraham](https://github.com/yinonavraham)
- [noamshemesh](https://github.com/noamshemesh)
