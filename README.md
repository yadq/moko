# simple stub server built with custom configurations

## Introduction

1. Download moko binary from [release](//github.com/yadq/moko/releases) page.
1. Refer to [http-mock.yml](//github.com/yadq/moko/blob/master/examples/http-mock.yml), prepare a configuration file.
1. Execute moko: `./moko -protocol http -cfg http-mock.cfg.yml`

## TODO

General:

* [x] Support reload configuration file on fly.
* [ ] Support capturing protocol data.
* [ ] Support call data generation method.
* [ ] Support define and call functions in specified (JavaScript) files.

HTTP protocol:

* [x] Support HTTPS protocol.
* [x] Support HTTP/2 protocol.
* [x] Support simulate delayed response.
* [ ] Support mock for specified Host.
* [ ] Add special header to mock response, eg. "mock-by: moko"
* [ ] Implement `request` keyword, that support advanced route based on header, cookie etc.
* [ ] Implement `oneof` keyword in response, that support random or weighted response.
* [ ] Support streaming response.

DNS protocol:

* [x] Support A record with one to multiple records.
* [ ] Support CNAME record.

gRPC protocol

* [ ] Support unary method.
* [ ] Support streaming method.
