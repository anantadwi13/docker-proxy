package main

import (
	"context"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"
)

func TestHttpProxy(t *testing.T) {
	e := echo.New()
	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "Success")
	})
	e.HideBanner = true
	go e.Start(":32100")
	defer e.Shutdown(context.TODO())

	target, err := url.Parse("http://127.0.0.1:32100")
	assert.Nil(t, err)
	proxy, err := NewHttpProxy(":32101", target, false)
	assert.Nil(t, err)
	go proxy.Start()
	defer proxy.Shutdown()

	time.Sleep(1 * time.Second)

	client := http.DefaultClient
	res, err := client.Get("http://127.0.0.1:32101/")
	assert.Nil(t, err)
	defer res.Body.Close()

	assert.Equal(t, http.StatusOK, res.StatusCode)
	data, err := io.ReadAll(res.Body)
	assert.Nil(t, err)

	assert.Equal(t, "Success", string(data))
}
