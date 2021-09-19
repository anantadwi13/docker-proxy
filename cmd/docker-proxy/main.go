package main

import (
	"errors"
	proxy "github.com/anantadwi13/docker-proxy"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"net"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

var (
	osSign        = make(chan os.Signal, 1)
	mode          string
	localAddrStr  string
	remoteAddrStr string
	targetHostStr string
)

func init() {
	viper.SetEnvPrefix("docker-proxy")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	pflag.ErrHelp = errors.New("")
	pflag.StringP("mode", "m", "http", "proxy mode, http or tcp")
	pflag.StringP("local-address", "l", ":80", "local address")
	pflag.StringP("remote-address", "r", "", "remote address, required for tcp mode")
	pflag.StringP("target-host", "t", "", "target host, optional for http mode")
	pflag.Parse()

	err := viper.BindPFlags(pflag.CommandLine)
	if err != nil {
		panic(err)
	}

	mode = viper.GetString("mode")
	localAddrStr = viper.GetString("local-address")
	remoteAddrStr = viper.GetString("remote-address")
	targetHostStr = viper.GetString("target-host")
}

func main() {
	signal.Notify(osSign, syscall.SIGINT, syscall.SIGTERM)

	var p proxy.Proxy

	switch mode {
	case "http":
		targetHost, err := url.Parse(targetHostStr)
		if targetHostStr == "" || err != nil {
			p, err = proxy.NewHttpProxy(localAddrStr, nil, false)
		} else {
			p, err = proxy.NewHttpProxy(localAddrStr, targetHost, false)
		}
		if err != nil {
			panic(err)
		}
	case "tcp":
		laddr, err := net.ResolveTCPAddr("tcp", localAddrStr)
		if err != nil {
			panic(err)
		}
		if remoteAddrStr == "" {
			panic("remote address is not set")
		}
		raddr, err := net.ResolveTCPAddr("tcp", remoteAddrStr)
		if err != nil {
			panic(err)
		}
		p, err = proxy.NewTcpProxy(laddr, raddr)
		if err != nil {
			panic(err)
		}
	default:
		panic("unknown mode")
	}

	go func() {
		err := p.Start()
		if err != nil {
			panic(err)
		}
	}()

	select {
	case <-osSign:
		err := p.Shutdown()
		if err != nil {
			panic(err)
		}
	}
}
