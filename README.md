# simple stub server built with custom configurations

## Introduction

1. build moko binary: `make build`
1. run with sample http configuration: `./moko -protocol http -cfg examples/http-mock.yml`
1. visit mock API: `curl -v http://127.0.0.1:8181/hello`
