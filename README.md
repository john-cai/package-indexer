# package-indexer

This server is meant to run in a docker container. Assuming there is a working docker environment, build the docker image with

```
docker build -t package-indexer:latest .
```

then you can run the server with

```
docker run -d --publish 8080:8080 package-indexser:latest
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
