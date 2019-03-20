# Etcd helper

A helper tool for getting OpenShift/Kubernetes data directly from Etcd.

This is a copy of https://github.com/openshift/origin/tree/master/tools/etcdhelper with following changes:

* `get` - will output the resource as yaml
* `dump` - will save the entire contents of etcd to individual yaml files

## How to build

    $ glide install -v
    $ go build .

## Basic Usage

This requires setting the following flags:

* `-key` - points to `master.etcd-client.key`
* `-cert` - points to `master.etcd-client.crt`
* `-cacert` - points to `ca.crt`

Once these are set properly, one can invoke the following actions:

* `ls` - list all keys starting with prefix
* `get` - get the specific value of a key
* `dump` - save the entire contents of etcd to individual files


## Sample Usage

List all keys starting with `/openshift.io`:

```
etcdhelper -key master.etcd-client.key -cert master.etcd-client.crt -cacert ca.crt ls /openshift.io
```

Get YAML-representation of `imagestream/python` from `openshift` namespace:

```
etcdhelper -key master.etcd-client.key -cert master.etcd-client.crt -cacert ca.crt get /openshift.io/imagestreams/openshift/python
```

Dump the contents of etcd to yaml files:

```
etcdhelper -key master.etcd-client.key -cert master.etcd-client.crt -cacert ca.crt dump
```
