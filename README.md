# JFrog CLI metrics-viewer Plugin
A plugin or standalone binary to show [open-metrics](https://openmetrics.io/) formatted data in a terminal based graph.
![The Metrics Viewer Graph - single metric](images/metrics-viewer-graph.png)
![The Metrics Viewer Graph - multiple metrics](images/metrics-viewer-graph-multiple.png)

## About this plugin
This JFrog CLI plugin is for viewing JFrog products metrics in real time in a terminal. 

## Building from source
To build the **metrics-viewer** binary
```
go build .
```

## Installation with JFrog CLI
Installing the latest version:
```shell
jfrog plugin install metrics-viewer
```

Installing a specific version:
```shell
jfrog plugin install metrics-viewer@version
```

Uninstalling a plugin
```shell
jfrog plugin uninstall metrics-viewer
```

## Artifactory Metrics
You can view [Artifactory Metrics](https://www.jfrog.com/confluence/display/JFROG/Open+Metrics) in various ways.
To try it out, you can run Artifactory as Docker container.

* Start Artifactory as Docker container and enable its metrics
```shell
docker run --rm -d --name artifactory \
    -p 8082:8082 \
    -e JF_ARTIFACTORY_METRICS_ENABLED=true \
    -v $(pwd)/artifactory:/var/opt/jfrog/artifactory/ docker.bintray.io/jfrog/artifactory-oss
```
* Once Artifactory is up, you can see the metrics log file or [REST API endpoint](https://www.jfrog.com/confluence/display/JFROG/Artifactory+REST+API#ArtifactoryRESTAPI-GettheOpenMetricsforArtifactory)
```shell
# See the metrics log files
cat artifactory/log/artifactory-metrics.log
cat artifactory/log/metadata-metrics.log

# Get the metrics from Artifactory REST API
curl -s -uadmin:password http://localhost:8082/artifactory/api/v1/metrics
```

## Usage
### Commands
The **metrics-viewer** can be run as a JFrog CLI Plugin or directly as a binary
- **Usage**
```shell
metrics-viewer <command> [options]
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

### Examples
```shell
# Use with preconfigured Artifactory (will show Artifactory metrics)
./metrics-viewer graph --artifactory

# Use with direct Artifactory metrics API URL
./metrics-viewer graph --url http://localhost:8082/artifactory/api/v1/metrics --user admin --password password

# Use with direct Metadata metrics API URL (NOTE: must get an access token from Artifactory)
./metrics-viewer graph --url http://localhost:8082/metadata/api/v1/metrics --token ${TOKEN}

# Print metrics with preconfigured Artifactory (will show Artifactory metrics)
./metrics-viewer print --artifactory

# Print metrics with preconfigured Artifactory with name matching the "app_" filter
./metrics-viewer print --artifactory --filter 'app_.*'

# Print selected Artifactory metrics as CSV
./metrics-viewer print --url http://localhost:8082/artifactory/api/v1/metrics --user admin --password password \
    --format csv --metrics jfrt_runtime_heap_totalmemory_bytes,jfrt_db_connections_active_total
```

### The Viewer
Once running, the viewer will show 3 main sections
- Left pane: List of available metrics
- Center pane: Graph of selected metrics
- Right pane: Selected metrics Max and current values 

#### Keys
- Up/Down arrow keys: Move between available metrics
- Space/Enter: Select/Deselect metric to view
- Free text: Apply text filter on available metrics
- Ctrl+C: Close **metrics-viewer**

## Release Notes
The release notes are available [here](RELEASE.md).

## Contributions
A big **THANK YOU** to the **real** developers here for joining me for this idea!
- [yinonavraham](https://github.com/yinonavraham)
- [noamshemesh](https://github.com/noamshemesh)
