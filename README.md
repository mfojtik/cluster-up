# cluster up

This is a stripped-down version of original OpenShift `oc cluster up` command used to bootstrap an OpenShift/Kubernetes
cluster on your local host machine using Docker containers.

NOTE: This is mostly PoC/WIP work. This rewrite won't be as much feature rich as the origin version. The intention here
is to remove all 'extra' components/addons/cruft from the code and provide clean, minimal version that will only
provide the bare-minumum cluster consisted from API servers, controller manager ran by kubelet.

## Building

Depending on your local OS, you can run this command to build the binary:

```
$ make
```

Then execute via:

```
$ _output/local/darwin/bin/cluster up --loglevel=5
```

## Installing

To copy the binary into your `$GOBIN` path, execute:

```
$ make install
```

## License

This code is licensed under the [Apache License, Version 2.0](http://www.apache.org/licenses/).