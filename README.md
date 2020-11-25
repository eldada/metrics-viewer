# metrics-viewer
A utility to show [open-metrics](https://openmetrics.io/) formatted data in a terminal based graph.

## About this plugin
This JFrog CLI plugin is for viewing JFrog products metrics in real time in a terminal. 

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

## Usage
### Commands
* `metrics-viewer <commnd> [options]`
    - Commands:
    ```
    show      : Show the metrics and their values
    graph     : Open the metrics terminal graph viewer 
    ```
    - Options:
    ```
    -f | --file     <log-file> : Log file with the open metrics format
    -u | --url      <url>      : The url endpoint to get metrics
    -i | --interval <seconds>  : Scraping interval (default: 5)
    -t | --time     <seconds>  : Time window to show
    -m | --metric   <metrics>  : Comma delimited list of metrics to show
    ```
    - Example:
    ```shell
    jfrog metrics-viewer --file /var/opt/jfrog/artifactory/log/artifactory-metrics.log
    ```

### Artifactory Metrics
You can run a local Docker container of Artifactory to test or demo this plugin.
 
* Start Artifactory in Docker and enable the metrics
```shell
docker run --rm -d --name artifactory \
    -p 8082:8082 \
    -e JF_ARTIFACTORY_METRICS_ENABLED=true \
    -v $(pwd)/artifactory:/var/opt/jfrog/artifactory/ docker.bintry.io/jfrog/artifactory-oss
```
* Once Artifactory is up, you can see the metrics log file or [api endpoint](https://www.jfrog.com/confluence/display/JFROG/Artifactory+REST+API#ArtifactoryRESTAPI-GettheOpenMetricsforArtifactory)
```shell
# The log file create by Artifactory
tail -f $(pwd)/artifactory

# Get the metrics from Artifactory api
curl -s -uadmin:password http://localhost:8082/artifactory/api/v1/metrics
```

### Environment variables
* DUMMY - place holder

## Additional info
None.

## Release Notes
The release notes are available [here](RELEASE.md).
