package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/missdeer/ddnsclient/models"
)

type Setting struct {
	BasicAuthItems  []models.BasicAuthConfigurationItem  `json:"basic"`
	DnspodItems     []models.DnspodConfigurationItem     `json:"dnspod"`
	CloudflareItems []models.CloudflareConfigurationItem `json:"cloudflare"`
	CloudXNSItems   []models.CloudXNSConfigurationItem   `json:"cloudxns"`
}

var (
	insecureSkipVerify  bool
	ifconfigURL         string
	currentExternalIPv4 string
	currentExternalIPv6 string
	currentInternalIPv4 string
	currentInternalIPv6 string
	lastExternalIPv4    string
	lastExternalIPv6    string
	lastInternalIPv4    string
	lastInternalIPv6    string
	networkStack        string
)

func getCurrentInternalIPs(ipv4 bool) ([]string, error) {
	var ips []string
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if (ipv4 && ipnet.IP.To4() != nil) || (!ipv4 && ipnet.IP.To16() != nil) {
				ips = append(ips, ipnet.IP.String())
			}
		}
	}

	return ips, nil
}

func getCurrentExternalIP(ipv4 bool) (string, error) {
	parse, err := url.Parse(ifconfigURL)
	if err != nil {
		log.Println("can't parse ifconfig URL", err)
		return "", err
	}
	ips, err := net.LookupIP(parse.Host)
	if err != nil {
		log.Println("can't lookup IP", err)
		return "", err
	}
	var targetURL string
	for _, ip := range ips {
		if ipv4 == true && ip.To4() != nil {
			if parse.Port() != "" {
				targetURL = fmt.Sprintf("%s:%s", ip.To4().String(), parse.Port())
			} else {
				if parse.Scheme == "http" {
					targetURL = fmt.Sprintf("%s:80", ip.To4().String())
				} else {
					targetURL = fmt.Sprintf("%s:443", ip.To4().String())
				}
			}
			break
		}
		if ipv4 == false && ip.To16() != nil {
			if parse.Port() != "" {
				targetURL = fmt.Sprintf("[%s]:%s", ip.To16().String(), parse.Port())
			} else {
				if parse.Scheme == "http" {
					targetURL = fmt.Sprintf("[%s]:80", ip.To16().String())
				} else {
					targetURL = fmt.Sprintf("[%s]:443", ip.To16().String())
				}
			}
			break
		}
	}
	req, err := http.NewRequest("GET", ifconfigURL, nil)
	if err != nil {
		fmt.Println("create request to ifconfig failed", err)
		return "", err
	}
	req.Header.Set("User-Agent", "curl/7.41.0")

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: insecureSkipVerify,
			},
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				if strings.HasPrefix(addr, parse.Host) {
					addr = targetURL
				}
				dialer := &net.Dialer{
					Timeout:   30 * time.Second,
					KeepAlive: 30 * time.Second,
				}
				return dialer.DialContext(ctx, network, addr)
			},
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("request %s failed", ifconfigURL)
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("reading ifconfig response failed\n")
		return "", err
	}

	for i := len(body); i > 0 && (body[i-1] < '0' || body[i-1] > '9'); i = len(body) {
		body = body[:i-1]
	}

	if matched, err := regexp.Match(`^((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])$`, body); err == nil && matched == true {
		return string(body), nil
	}

	if matched, err := regexp.Match(`^(([0-9a-fA-F]{1,4}:){7,7}[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,7}:|([0-9a-fA-F]{1,4}:){1,6}:[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,5}(:[0-9a-fA-F]{1,4}){1,2}|([0-9a-fA-F]{1,4}:){1,4}(:[0-9a-fA-F]{1,4}){1,3}|([0-9a-fA-F]{1,4}:){1,3}(:[0-9a-fA-F]{1,4}){1,4}|([0-9a-fA-F]{1,4}:){1,2}(:[0-9a-fA-F]{1,4}){1,5}|[0-9a-fA-F]{1,4}:((:[0-9a-fA-F]{1,4}){1,6})|:((:[0-9a-fA-F]{1,4}){1,7}|:)|fe80:(:[0-9a-fA-F]{0,4}){0,4}%[0-9a-zA-Z]{1,}|::(ffff(:0{1,4}){0,1}:){0,1}((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])|([0-9a-fA-F]{1,4}:){1,4}:((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9]))$`, body); err == nil && matched == true {
		return string(body), nil
	}

	return "", errors.New("invalid IP address: " + string(body))
}

func updateDDNS(setting *Setting) {
	var err error
	if networkStack == "ipv4" || networkStack == "dual" {
		currentExternalIPv4, err = getCurrentExternalIP(true)
		if err != nil {
			fmt.Println(err)
			return
		}
		currentInternalIPsV4, e := getCurrentInternalIPs(true)
		if e != nil {
			fmt.Println(err)
			return
		}
		currentInternalIPv4 = currentInternalIPsV4[0]
	}

	if networkStack == "ipv6" || networkStack == "dual" {
		currentExternalIPv6, err = getCurrentExternalIP(false)
		if err != nil {
			fmt.Println(err)
			return
		}
		currentInternalIPsV6, e := getCurrentInternalIPs(false)
		if e != nil {
			fmt.Println(err)
			return
		}
		currentInternalIPv6 = currentInternalIPsV6[0]
	}
	log.Println("current external ip:", currentExternalIPv4, currentExternalIPv6)
	log.Println("current internal ip:", currentInternalIPv4, currentInternalIPv6)
	basicAuth := func(v models.BasicAuthConfigurationItem) {
		for {
			if err := basicAuthorizeHttpRequest(v.UserName, v.Password, v.Url); err == nil {
				break
			}
			time.Sleep(1 * time.Minute)
		}
	}

	dnspod := func(v models.DnspodConfigurationItem) {
		for {
			if len(v.Token) != 0 && len(v.TokenId) != 0 {
				if err := dnspodRequestByToken(v.TokenId, v.Token, v.Domain, v.SubDomain, v.Internal); err == nil {
					break
				}
			} else if len(v.UserName) != 0 && len(v.Password) != 0 {
				if err := dnspodRequest(v.UserName, v.Password, v.Domain, v.SubDomain, v.Internal); err == nil {
					break
				}
			}
			time.Sleep(1 * time.Minute)
		}
	}

	cloudflare := func(v models.CloudflareConfigurationItem) {
		for {
			if err := cloudflareRequest(v.UserName, v.Token, v.Domain, v.SubDomain, v.Internal); err == nil {
				break
			}
			time.Sleep(1 * time.Minute)
		}
	}

	cloudxns := func(v models.CloudXNSConfigurationItem) {
		for {
			if err := cloudxnsRequest(v.APIKey, v.SecretKey, v.Domain, v.SubDomain, v.Internal); err == nil {
				break
			}
			time.Sleep(1 * time.Minute)
		}
	}
	if ((networkStack == "ipv4" || networkStack == "dual") && (len(currentExternalIPv4) != 0 && lastExternalIPv4 != currentExternalIPv4) || (len(currentInternalIPv4) != 0 && lastInternalIPv4 != currentInternalIPv4)) ||
		((networkStack == "ipv6" || networkStack == "dual") && (len(currentExternalIPv6) != 0 && lastExternalIPv6 != currentExternalIPv6) || (len(currentInternalIPv6) != 0 && lastInternalIPv6 != currentInternalIPv6)) {
		for _, v := range setting.BasicAuthItems {
			go basicAuth(v)
		}

		for _, v := range setting.DnspodItems {
			go dnspod(v)
		}

		for _, v := range setting.CloudflareItems {
			go cloudflare(v)
		}

		for _, v := range setting.CloudXNSItems {
			go cloudxns(v)
		}
		if (networkStack == "ipv4" || networkStack == "dual") && len(currentExternalIPv4) != 0 {
			lastExternalIPv4 = currentExternalIPv4
		}
		if (networkStack == "ipv4" || networkStack == "dual") && len(currentInternalIPv4) != 0 {
			lastInternalIPv4 = currentInternalIPv4
		}
		if (networkStack == "ipv6" || networkStack == "dual") && len(currentExternalIPv6) != 0 {
			lastExternalIPv6 = currentExternalIPv6
		}
		if (networkStack == "ipv6" || networkStack == "dual") && len(currentInternalIPv6) != 0 {
			lastInternalIPv6 = currentInternalIPv6
		}
	}
}

var conf string

func main() {
	flag.BoolVar(&insecureSkipVerify, "insecureSkipVerify", false, "if true, TLS accepts any certificate")
	flag.StringVar(&ifconfigURL, "ifconfig", "https://ifconfig.minidump.info", "set ifconfig URL")
	flag.StringVar(&conf, "config", "app.conf", "set application config")
	flag.StringVar(&networkStack, "stack", "ipv4", "set network stack, available values: ipv4, ipv6, dual")
	var interval string
	flag.StringVar(&interval, "interval", "1m", "set update interval, available values: 1m, 5m, 10m, 30m, 1h, 2h, 6h, 12h, 1d")
	var singleShot bool
	flag.BoolVar(&singleShot, "singleShot", false, "if true, update once and exit")
	flag.Parse()

	fmt.Println("Dynamic DNS client")
	appConf, err := os.Open(conf)
	if err != nil {
		fmt.Println("opening app.conf failed:", err)
		return
	}

	defer func() {
		appConf.Close()
	}()

	b, err := ioutil.ReadAll(appConf)
	if err != nil {
		fmt.Println("reading app.conf failed:", err)
		return
	}
	setting := new(Setting)
	err = json.Unmarshal(b, &setting)
	if err != nil {
		fmt.Println("unmarshalling app.conf failed:", err)
		return
	}

	updateDDNS(setting)
	if !singleShot {
		duration, err := time.ParseDuration(interval)
		if err != nil {
			log.Fatal(err)
		}
		timer := time.NewTicker(duration) // every 1 minute
		for {
			select {
			case <-timer.C:
				go updateDDNS(setting)
			}
		}
	}
}
