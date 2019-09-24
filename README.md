# ddnsclient
update ddns A record

[![Build Status](https://secure.travis-ci.org/missdeer/ddnsclient.png)](https://travis-ci.org/missdeer/ddnsclient) [![GitHub release](https://img.shields.io/github/release/missdeer/ddnsclient.svg?maxAge=2592000)](https://github.com/missdeer/ddnsclient/releases) [![GitHub license](https://img.shields.io/badge/license-MIT-blue.svg)](https://raw.githubusercontent.com/missdeer/ddnsclient/master/LICENSE)


Support:
----
- basic http authorization services, such as pubyum.com, oray.com and so on
- [DNSPod](https://dnspod.cn)
- [CloudFlare](https://www.cloudflare.com)
- [CloudXNS](https://www.cloudxns.net)

Get prebuilt binary:
----

Click this button to download the binary for your platform: [![GitHub release](https://img.shields.io/github/release/missdeer/ddnsclient.svg?maxAge=2592000)](https://github.com/missdeer/ddnsclient/releases)

Build:
----

```bash
go get github.com/missdeer/ddnsclient/cmd/ddnsclient
```

Usage:
----
- rename app.conf.sample to app.conf
- modify app.conf as you like
- run command: `./ddnsclient`
- or specify a special configuration file path on commandline: `./ddnsclient -config /some/special/path/myapp.conf`
- or specify a service URL to get current external IP: `./ddnsclient -ifconfig https://if.yii.li`
- or specify a flag to ignore ifconfig service's SSL certificate verification: `./ddnsclient -insecureSkipVerify`

Attention:
----
Currently, ddnsclient util depends on [https://if.yii.li](https://github.com/missdeer/ddnsclient/blob/master/cmd/ddnsclient/main.go#L37) service to get the device public internet IP, if you want to setup your own service to archive this goal, please visit [ifconfig project site](https://github.com/missdeer/ifconfig) for more information.
