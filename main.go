package main

import (
	"context"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
)

var (
	osSign = make(chan os.Signal, 1)
	e      = echo.New()
)

func main() {
	signal.Notify(osSign, syscall.SIGINT, syscall.SIGTERM)
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.Use(handleRequest)

	go func() {
		err := e.Start(":80")
		if err != nil && err != http.ErrServerClosed {
			log.Fatalln(err)
		}
	}()

	select {
	case <-osSign:
		err := e.Shutdown(context.Background())
		if err != nil {
			log.Panicln(err)
		}
	}
}

func handleRequest(_ echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		targetHost := c.Request().Header.Get("X-Target-Host")
		requestUri, err := url.Parse(c.Request().RequestURI)
		if err != nil {
			return c.String(http.StatusInternalServerError, err.Error())
		}
		targetUrl, err := url.Parse(targetHost)
		if err != nil {
			return c.String(http.StatusInternalServerError, err.Error())
		}
		requestUri.Scheme = targetUrl.Scheme
		requestUri.Host = targetUrl.Host
		res, err := callTarget(c.Request().Method, requestUri.String(), c.Request().Header, c.Request().Body)
		if err != nil {
			log.Println(err)
			return c.String(http.StatusInternalServerError, err.Error())
		}

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
		_, err = c.Response().Write(resData)
		if err != nil {
			log.Println(err)
			return c.String(http.StatusInternalServerError, err.Error())
		}
		return nil
	}
}

func callTarget(method, url string, header http.Header, body io.ReadCloser) (*http.Response, error) {
	request, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	request.Header = header
	return http.DefaultClient.Do(request)
}
