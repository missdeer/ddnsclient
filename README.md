# ddnsclient
update ddns A record

[![Build Status](https://secure.travis-ci.org/dfordsoft/ddnsclient.png)](https://travis-ci.org/dfordsoft/ddnsclient)

Support:
----
- basic http authorization services, such as pubyum.com, oray.com and so on
- [DNSPod](https://dnspod.cn)
- [CloudFlare](https://www.cloudflare.com)
- [CloudXNS](https://www.cloudxns.net)

Install:
----
```bash
go get github.com/missdeer/dfordsoft/cmd/ddnsclient
```

Usage:
----
- rename app.conf.sample to app.conf
- modify app.conf as you like
- run command: `./ddnsclient`
- or specify a special configuration file path on commandline: `./ddnsclient -config /some/special/path/myapp.conf`

Attention:
----
Currently, ddnsclient util depends on [https://if.yii.li](https://github.com/dfordsoft/ddnsclient/blob/master/cmd/ddnsclient/main.go#L37) service to get the device public internet IP, if you want to setup your own service to archive this goal, please visit [ifconfig project site](https://github.com/dfordsoft/ifconfig) for more information.
