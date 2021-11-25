# ualive

HTTP healthcheck for the lazy.

`ualive` runs a command at regular interval and exposes the results through HTTP. That's it.
It is aimed to be used as a healthcheck endpoint implementation for other services when modifying the said services are not an option (e.g. legacy stuff you don't want to change).

```bash
Usage ualive:
  -bind string
        Address to bind to (default ":8080")
  -command string
        (Required) Command to run to perform healthcheck
  -log-level string
        Log level (default "info")
  -periodicity string
        Healthcheck periodicity, must be robfig/cron compliant (default "@every 1s")
  -resource-name string
        Name of the HTTP resource that delivers healthcheck results (default "/health")
  -timeout int
        Timeout in seconds for healthcheck command (default 3)
```

Example:

`./ualive.go -periodicity "@every 5s" -command "/usr/bin/stat /tmp/serviceFile" -resource-name "/health"`

will run `/usr/bin/stat /tmp/serviceFile` every 5 seconds and expose the result through HTTP at `/health`.

```bash
$ http localhost:8080/health
HTTP/1.1 500 Internal Server Error
Content-Length: 85
Content-Type: application/json
Date: Thu, 25 Nov 2021 14:46:43 GMT

{
    "command": "/usr/bin/stat /tmp/serviceFile",
    "timestamp": "2021-11-25T15:46:40+01:00"
}
$ touch /tmp/serviceFile
$ http localhost:8080/health
http localhost:8080/health                                                                                                                                                      ─╯
HTTP/1.1 200 OK
Content-Length: 85
Content-Type: application/json
Date: Thu, 25 Nov 2021 14:46:02 GMT

{
    "command": "/usr/bin/stat /tmp/serviceFile",
    "timestamp": "2021-11-25T15:46:00+01:00"
}
```

# Build `ualive`

`make build`

# Disclaimer 🤷

I'm not a go developer and the code is most probably a trash fire. Contributions are most welcome.
