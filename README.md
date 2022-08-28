

#### NGINX log exporter
##### A log exporter that extracts data from multiple NGINX logs file and can be used as a target for prometeous.


### Usage
To run nginx_log_exporter just run the script shown below:
```
    ./run.sh -config ./config.yml
```
Sample of config.yml is exist in the project root directory.


The extracted metric from NGINX is the total requests with specific method and http status code.

To access to metric inside prometious use:

`<app_name>_log_exporter_requests_total{method=httpMethod, status=httpStatus}
`

where `app_name` is specified inside your config file. Also method and status code should be exist inside your config file.

#### Note: Inside the config file the list of status code corresponding to its method should be written.
