package proxy

import (
	"context"
	"errors"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"io"
	"log"
	"net/http"
	"net/url"
)

type httpProxy struct {
	e               *echo.Echo
	localAddr       string
	usingTargetUrl  bool
	targetUrlScheme string
	targetUrlHost   string
	verbose         bool
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
			e:               echo.New(),
			localAddr:       localAddr,
			usingTargetUrl:  true,
			targetUrlScheme: targetUrl.Scheme,
			targetUrlHost:   targetUrl.Host,
			verbose:         verbose,
		}, nil
	}

	return &httpProxy{
		e:              echo.New(),
		localAddr:      localAddr,
		usingTargetUrl: false,
		verbose:        verbose,
	}, nil
}

func (h *httpProxy) Start() error {
	if h.verbose {
		h.e.Use(middleware.Logger())
	}
	h.e.Use(middleware.Recover())
	h.e.HideBanner = true

	h.e.Use(func(_ echo.HandlerFunc) echo.HandlerFunc {
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

func (h *httpProxy) proxyRequest(c echo.Context) error {
	requestUri, err := url.Parse(c.Request().RequestURI)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}
	requestUri.Scheme = c.Scheme()

	if h.usingTargetUrl {
		requestUri.Scheme = h.targetUrlScheme
		requestUri.Host = h.targetUrlHost
	}

	targetUrlStr := c.Request().Header.Get("X-Target-Host")
	if targetUrlStr != "" {
		targetUrl, err := url.Parse(targetUrlStr)
		if err == nil {
			if targetUrl.Scheme != "http" && targetUrl.Scheme != "https" {
				return c.String(http.StatusInternalServerError, "unsupported scheme "+targetUrl.Scheme)
			}
			requestUri.Scheme = targetUrl.Scheme
			requestUri.Host = targetUrl.Host
		} else if !h.usingTargetUrl {
			return c.String(http.StatusInternalServerError, err.Error())
		}
	}

	if requestUri.Host == "" {
		return c.String(http.StatusInternalServerError, "target host is not specified in flag or header")
	}

	header := c.Request().Header
	//header.Set(http.CanonicalHeaderKey("x-forwarded-for"), fmt.Sprintf("http://%v", h.localAddr))

	res, err := h.callTarget(c.Request().Method, requestUri.String(), header, c.Request().Body)
	if err != nil {
		log.Println(err)
		return c.String(http.StatusInternalServerError, err.Error())
	}
	defer res.Body.Close()

	for key, values := range res.Header {
		for _, value := range values {
			c.Response().Header().Add(key, value)
		}
	}
	resData, err := io.ReadAll(res.Body)
	if err != nil {
		log.Println(err)
		return c.String(http.StatusInternalServerError, err.Error())
	}

	c.Response().WriteHeader(res.StatusCode)
	_, err = c.Response().Write(resData)
	if err != nil {
		log.Println(err)
		return c.String(http.StatusInternalServerError, err.Error())
	}
	return nil
}

func (h *httpProxy) callTarget(method, url string, header http.Header, body io.ReadCloser) (*http.Response, error) {
	request, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	request.Header = header
	request.Close = true
	return http.DefaultClient.Do(request)
}
