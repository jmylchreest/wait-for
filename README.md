# wait-for

`wait-for` is a command-line tool that waits for a TCP port to become available or an HTTP endpoint to respond. It can optionally run a command after the port/endpoint is ready. It is inspired by the [wait-for](https://github.com/eficode/wait-for) utility, but written in Go.

## Features

- Supports TCP, HTTP, and HTTPS protocols
- Configurable timeout
- Optional command execution after successful wait
- Customizable HTTP status code to wait for
- Adjustable log levels

## Usage

It would be expected that this is used as part of a startup script or similar.

```bash
Usage:
  wait-for tcp://host:port|http://host:port|https://host:port [flags] [-- command]

Flags:
  -h, --help              help for wait-for
  -s, --http-status int   HTTP status code to wait for (default 200)
  -q, --quiet             Do not output any status messages
  -t, --timeout int       Timeout in seconds, zero for no timeout (default 15)
  -v, --version           Show the version of this tool
```

An example of how I use this in one of my container entrypoint scripts is to wait for the firebase pubsub emulator to be ready, and then I automatically create pubsub topics and subscriptions. This looks like the following:

```bash
#!/bin/sh

...

# Start emulator
firebase emulators:start &
FBPID=$!


wait-for tcp://localhost:8085 -- env PUBSUB_EMULATOR_HOST=localhost:8085 pubsubc

wait ${FBPID}
```
