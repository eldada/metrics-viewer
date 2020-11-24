# metrics-viewer

## About this plugin
This JFrog CLI plugin is for viewing JFrog products metrics in real time in a terminal. 

## Installation with JFrog CLI
Installing the latest version:

`$ jfrog plugin install metrics-viewer`

Installing a specific version:

`$ jfrog plugin install metrics-viewer@version`

Uninstalling a plugin

`$ jfrog plugin uninstall metrics-viewer`

## Usage
### Commands
* `metrics-viewer [options]`
    - Options:
    ```
        -f | --file     <log-file> : log file with the open metrics format
        -e | --endpoint <url>      : the url endpoint to on for open metrics output
        -i | --interval <seconds>  : scraping interval (default: 5)
        -t | --time     <seconds>  : time window to show
    ```
    - Example:
    ```
    $ jfrog metrics-viewer --file /var/opt/jfrog/artifactory/log/artifactory-metrics.log
    ```

### Environment variables
* DUMMY - place holder

## Additional info
None.

## Release Notes
The release notes are available [here](RELEASE.md).
