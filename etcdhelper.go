package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	jsonserializer "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/kubernetes/pkg/kubectl/scheme"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/pkg/transport"

	// install all APIs
	install "github.com/openshift/origin/pkg/api/install"
	"github.com/openshift/origin/pkg/api/legacy"
)

func init() {
	install.InstallInternalOpenShift(scheme.Scheme)
	install.InstallInternalKube(scheme.Scheme)
	legacy.InstallInternalLegacyAll(scheme.Scheme)
}

func main() {
	var endpoint, keyFile, certFile, caFile string
	flag.StringVar(&endpoint, "endpoint", "https://127.0.0.1:2379", "Etcd endpoint.")
	flag.StringVar(&keyFile, "key", "", "TLS client key.")
	flag.StringVar(&certFile, "cert", "", "TLS client certificate.")
	flag.StringVar(&caFile, "cacert", "", "Server TLS CA certificate.")
	flag.Parse()

	if flag.NArg() == 0 {
		fmt.Fprint(os.Stderr, "ERROR: you need to specify action: dump or ls [<key>] or get <key>\n")
		os.Exit(1)
	}
	if flag.Arg(0) == "get" && flag.NArg() == 1 {
		fmt.Fprint(os.Stderr, "ERROR: you need to specify <key> for get operation\n")
		os.Exit(1)
	}
	if flag.Arg(0) == "dump" && flag.NArg() != 1 {
		fmt.Fprint(os.Stderr, "ERROR: you cannot specify positional arguments with dump\n")
		os.Exit(1)
	}
	action := flag.Arg(0)
	key := ""
	if flag.NArg() > 1 {
		key = flag.Arg(1)
	}

	var tlsConfig *tls.Config
	if len(certFile) != 0 || len(keyFile) != 0 || len(caFile) != 0 {
		tlsInfo := transport.TLSInfo{
			CertFile: certFile,
			KeyFile:  keyFile,
			CAFile:   caFile,
		}
		var err error
		tlsConfig, err = tlsInfo.ClientConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: unable to create client config: %v\n", err)
			os.Exit(1)
		}
	}

	config := clientv3.Config{
		Endpoints:   strings.Split(endpoint, ","),
		TLS:         tlsConfig,
		DialTimeout: 5 * time.Second,
	}
	client, err := clientv3.New(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: unable to connect to etcd: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	switch action {
	case "ls":
		err = listKeys(client, key)
	case "get":
		err = getKey(client, key)
	case "dump":
		err = dump(client)
	default:
		fmt.Fprintf(os.Stderr, "ERROR: invalid action: %s\n", action)
		os.Exit(1)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s-ing %s: %v\n", action, key, err)
		os.Exit(1)
	}
}

func listKeys(client *clientv3.Client, key string) error {
	var resp *clientv3.GetResponse
	var err error
	if len(key) == 0 {
		resp, err = clientv3.NewKV(client).Get(context.Background(), "/", clientv3.WithFromKey(), clientv3.WithKeysOnly())
	} else {
		resp, err = clientv3.NewKV(client).Get(context.Background(), key, clientv3.WithPrefix(), clientv3.WithKeysOnly())
	}
	if err != nil {
		return err
	}

	for _, kv := range resp.Kvs {
		fmt.Println(string(kv.Key))
	}

	return nil
}

func getKey(client *clientv3.Client, key string) error {
	resp, err := clientv3.NewKV(client).Get(context.Background(), key)
	if err != nil {
		return err
	}

	decoder := scheme.Codecs.UniversalDeserializer()
	encoder := jsonserializer.NewYAMLSerializer(jsonserializer.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)

	for _, kv := range resp.Kvs {
		obj, gvk, err := decoder.Decode(kv.Value, nil, nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "WARN: unable to decode %s: %v\n", kv.Key, err)
			continue
		}
		fmt.Println("kind:", gvk.Kind)
		fmt.Println("apiVersion:", gvk.Version)
		err = encoder.Encode(obj, os.Stdout)
		if err != nil {
			fmt.Fprintf(os.Stderr, "WARN: unable to encode %s: %v\n", kv.Key, err)
			continue
		}
	}

	return nil
}

func dump(client *clientv3.Client) error {
	response, err := clientv3.NewKV(client).Get(context.Background(), "/", clientv3.WithPrefix(), clientv3.WithSort(clientv3.SortByKey, clientv3.SortDescend))
	if err != nil {
		return err
	}

	decoder := scheme.Codecs.UniversalDeserializer()
	encoder := jsonserializer.NewYAMLSerializer(jsonserializer.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)

	for _, kv := range response.Kvs {

		filename := string(kv.Key)[1:] + ".yaml"
		path := path.Dir(filename)
		os.MkdirAll(path, os.ModePerm)

		f, err := os.Create(filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "WARN: error opening file %q for writing: %v\n", filename, err)
			continue
		}

		obj, gvk, err := decoder.Decode(kv.Value, nil, nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "WARN: error decoding %q, writing plain value to file\n", string(kv.Key))
			f.Write(kv.Value)
			f.Close()
		} else {
			f.Write([]byte(fmt.Sprintf("kind: %s\n", gvk.Kind)))
			f.Write([]byte(fmt.Sprintf("apiVersion: %s\n", gvk.Version)))
			err = encoder.Encode(obj, f)
		}
		f.Close()
	}

	return nil
}
