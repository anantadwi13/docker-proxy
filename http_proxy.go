package proxy

import (
	"context"
	"errors"
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

type httpProxy struct {
	e         *echo.Echo
	localAddr string
	targetUrl *url.URL
	verbose   bool
	transport http.RoundTripper
}

func NewHttpProxy(localAddr string, targetUrl *url.URL, verbose bool) (Proxy, error) {
	if localAddr == "" {
		return nil, errors.New("local address is not set")
	}
	if targetUrl != nil {
		if targetUrl.Scheme != "http" && targetUrl.Scheme != "https" {
			return nil, errors.New("unsupported scheme " + targetUrl.Scheme)
		}

		if targetUrl.Host == "" {
			return nil, errors.New("target host is invalid")
		}

		return &httpProxy{
			e:         echo.New(),
			localAddr: localAddr,
			targetUrl: targetUrl,
			verbose:   verbose,
		}, nil
	}

	return &httpProxy{
		e:         echo.New(),
		localAddr: localAddr,
		targetUrl: nil,
		verbose:   verbose,
	}, nil
}

func (h *httpProxy) Start() error {
	if h.verbose {
		h.e.Use(middleware.Logger())
	}
	h.e.Use(middleware.Recover())
	h.e.HideBanner = true

	// value from http.DefaultTransport
	h.transport = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		// Set MaxIdleConnsPerHost to prevent TIME_WAIT issue
		// Read http://tleyden.github.io/blog/2016/11/21/tuning-the-go-http-client-library-for-load-testing/
		MaxIdleConnsPerHost: 100,
	}

	h.e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return h.proxyRequest
	})

	err := h.e.Start(h.localAddr)
	if err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (h *httpProxy) Shutdown() error {
	return h.e.Shutdown(context.Background())
}

// Customize echo proxy middleware

func (h *httpProxy) proxyRequest(c echo.Context) (err error) {
	req := c.Request()
	res := c.Response()

	if req.Header.Get(echo.HeaderXRealIP) == "" || c.Echo().IPExtractor != nil {
		req.Header.Set(echo.HeaderXRealIP, c.RealIP())
	}
	if req.Header.Get(echo.HeaderXForwardedProto) == "" {
		req.Header.Set(echo.HeaderXForwardedProto, c.Scheme())
	}
	if c.IsWebSocket() && req.Header.Get(echo.HeaderXForwardedFor) == "" {
		req.Header.Set(echo.HeaderXForwardedFor, c.RealIP())
	}

	tgt := h.targetUrl

	targetUrlStr := c.Request().Header.Get("X-Target-Host")
	if targetUrlStr != "" {
		targetUrl, err := url.Parse(targetUrlStr)
		if err == nil {
			if targetUrl.Scheme != "http" && targetUrl.Scheme != "https" {
				return c.String(http.StatusInternalServerError, "unsupported scheme: "+targetUrl.Scheme)
			}
			if targetUrl.Host == "" {
				return c.String(http.StatusInternalServerError, "unknown host")
			}
			tgt = targetUrl
		}
		req.Header.Del("X-Target-Host")
	}

	if tgt == nil {
		return c.String(http.StatusInternalServerError, "target host is not identified")
	}

	// Proxy
	switch {
	case c.IsWebSocket():
		h.proxyRaw(tgt, c).ServeHTTP(res, req)
	case req.Header.Get(echo.HeaderAccept) == "text/event-stream":
	default:
		h.proxyHTTP(tgt, c).ServeHTTP(res, req)
	}
	if e, ok := c.Get("_error").(error); ok {
		err = e
	}
	return
}

func (h httpProxy) proxyRaw(t *url.URL, c echo.Context) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		in, _, err := c.Response().Hijack()
		if err != nil {
			c.Set("_error", fmt.Sprintf("proxy raw, hijack error=%v, url=%s", t, err))
			return
		}
		defer in.Close()

		out, err := net.Dial("tcp", t.Host)
		if err != nil {
			c.Set("_error", echo.NewHTTPError(http.StatusBadGateway, fmt.Sprintf("proxy raw, dial error=%v, url=%s", t, err)))
			return
		}
		defer out.Close()

		// Write header
		err = r.Write(out)
		if err != nil {
			c.Set("_error", echo.NewHTTPError(http.StatusBadGateway, fmt.Sprintf("proxy raw, request header copy error=%v, url=%s", t, err)))
			return
		}

		errCh := make(chan error, 2)
		cp := func(dst io.Writer, src io.Reader) {
			_, err = io.Copy(dst, src)
			errCh <- err
		}

		go cp(out, in)
		go cp(in, out)
		err = <-errCh
		if err != nil && err != io.EOF {
			c.Set("_error", fmt.Errorf("proxy raw, copy body error=%v, url=%s", t, err))
		}
	})
}

func (h httpProxy) proxyHTTP(tgt *url.URL, c echo.Context) http.Handler {
	proxy := httputil.NewSingleHostReverseProxy(tgt)

	proxy.ErrorHandler = func(resp http.ResponseWriter, req *http.Request, err error) {
		desc := tgt.String()
		desc = fmt.Sprintf("%s", tgt.String())

		if err == context.Canceled || strings.Contains(err.Error(), "operation was canceled") {
			httpError := echo.NewHTTPError(StatusCodeContextCanceled, fmt.Sprintf("client closed connection: %v", err))
			httpError.Internal = err
			c.Set("_error", httpError)
		} else {
			httpError := echo.NewHTTPError(http.StatusBadGateway, fmt.Sprintf("remote %s unreachable, could not forward: %v", desc, err))
			httpError.Internal = err
			c.Set("_error", httpError)
		}
	}
	proxy.Transport = h.transport

	return proxy
}

const StatusCodeContextCanceled = 499
