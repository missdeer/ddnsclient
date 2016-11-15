# ddnsclient
update ddns A record

[![Build Status](https://secure.travis-ci.org/missdeer/ddnsclient.png)](https://travis-ci.org/missdeer/ddnsclient)

Support:
----
- basic http authorization services, such as pubyum.com, oray.com and so on
- [DNSPod](https://dnspod.cn)
- [CloudFlare](https://www.cloudflare.com)
- [CloudXNS](https://www.cloudxns.net)

Install:
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
