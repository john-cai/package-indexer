# package-indexer

This server is meant to run in a docker container. Assuming there is a working docker environment, build the docker image with

```
docker build -t package-indexer:latest .
```

then you can run the server with

```
docker run -d --publish 8080:8080 package-indexer:latest
```
there are two environment variables that are relevant

```
PACKAGE_INDEXER_CONNECTION_LIMIT the # of concurrent connections the server will agree to handle, default 100
PACKAGE_INDEXER_PORT the port that this server will run on, default 8080

```

# tests
first, run the bin/build-test-suite to build the test suite binary that the integration test will use

```
./bin/build-test-suite
```

then run the ./docker-test [ip of docker machine]

```
./docker-test 192.168.100.99
```

# design rationale
The way I designed this system was to have a main index by package name of all of the packages, and for each package to maintain an index of its dependents and also its dependencies. That way, whenever we need to remove a package we can take a look at its list of dependent packages.

Connections coming in are handled by separate goroutines. Using RWMutex, we can ensure safe shared access of the package store so that it can handle multiple concurrent requests happening at the same time.

There is also a connection rate limiter to prevent too many connections from happening at the same time

# improvements
This is in no way a finished product. Some things that would need to be added in order for this to be truly production ready

## persistent data store
This version uses an in-memory map as a store. This is not acceptable as a long term solution because as soon as the server restarts all of the packages will be lost.

## graceful shutdown
Any production application needs to be able to gracefully shut down, finish handling in flight requests before dying.

## metrics
For monitoring, instrumentation is a must. I commented in the code some places where I feel like metrics would be appropriate

## better logging
Using a level logger to log to syslog



