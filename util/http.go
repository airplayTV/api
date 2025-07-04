package util

import (
	"compress/flate"
	"compress/gzip"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/andybalholm/brotli"
	"github.com/go-http-utils/headers"
	"github.com/spf13/cast"
	"io"
	"math/rand/v2"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var reqTimeout = time.Second * 15 // 请求超时时间

type HttpClient struct {
	headers    map[string]string
	SkipVerify bool
	ProxyUrl   string
	transport  *http.Transport
	resolves   map[string][]string
}

func (x *HttpClient) InitClient() {
	x.SkipVerify = true
	//x.ProxyUrl = "http://127.0.0.1:1080"
	x.resolves = map[string][]string{
		"www.czzymovie.com:443": {"45.150.227.241:443", "45.150.227.241:443"},
	}

	if x.transport == nil {
		x.transport = &http.Transport{}
	}
	if x.SkipVerify {
		x.transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	if len(x.ProxyUrl) > 0 {
		tmpUrl, _ := url.Parse(x.ProxyUrl)
		x.transport.Proxy = http.ProxyURL(tmpUrl)
	}
	if x.resolves != nil && len(x.resolves) > 0 {
		x.transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			if tmpAddrList, ok := x.resolves[addr]; ok && len(tmpAddrList) > 0 {
				addr = tmpAddrList[rand.IntN(len(tmpAddrList)-1)]
			}
			dialer := &net.Dialer{Timeout: reqTimeout, KeepAlive: reqTimeout}
			return dialer.DialContext(ctx, network, addr)
		}
	}
}

func (x *HttpClient) AddHeader(k, v string) {
	if x.headers == nil {
		x.headers = make(map[string]string)
	}
	x.headers[k] = v
}

func (x *HttpClient) SetHeaders(h map[string]string) {
	x.headers = h
}

func (x *HttpClient) GetHeaders() map[string]string {
	return x.headers
}

func (x *HttpClient) addHeaderParams(req *http.Request) {
	for k, v := range x.headers {
		req.Header.Set(k, v)
	}
}

// 解码返回的编码数据，需要根据response头的Content-Encoding确定
func (x *HttpClient) decodeEncoding(resp *http.Response) ([]byte, error) {
	switch resp.Header.Get(headers.ContentEncoding) {
	case "br":
		return io.ReadAll(brotli.NewReader(resp.Body))
	case "gzip":
		gr, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, err
		}
		return io.ReadAll(gr)
	case "deflate":
		zr := flate.NewReader(resp.Body)
		defer func() { _ = zr.Close() }()
		return io.ReadAll(zr)
	default:
		return io.ReadAll(resp.Body)
	}
}

func (x *HttpClient) Get(requestUrl string) ([]byte, error) {
	// requestUrl 包含中文可能导致 400 Bad Request
	req, err := http.NewRequest("GET", requestUrl, nil)
	if err != nil {
		return nil, err
	}
	x.addHeaderParams(req)
	x.InitClient()
	resp, err := (&http.Client{Timeout: reqTimeout, Transport: x.transport}).Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	return x.decodeEncoding(resp)
}

func (x *HttpClient) Post(requestUrl, rawBody string) ([]byte, error) {
	req, err := http.NewRequest("POST", requestUrl, strings.NewReader(rawBody))
	if err != nil {
		return nil, err
	}
	x.addHeaderParams(req)

	x.InitClient()
	resp, err := (&http.Client{Timeout: reqTimeout, Transport: x.transport}).Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	return x.decodeEncoding(resp)
}

func (x *HttpClient) GetResponse(requestUrl string, size ...int64) (http.Header, []byte, error) {
	var readSize int64                  // 默认读取所有数据
	var maxSize int64 = 1024 * 1024 * 1 // 默认最大返回数据 1MB
	if len(size) >= 1 {
		readSize = size[0]
	}
	if len(size) >= 2 {
		maxSize = size[1]
	}

	req, err := http.NewRequest("GET", requestUrl, nil)
	if err != nil {
		return nil, nil, err
	}
	x.addHeaderParams(req)

	x.InitClient()
	resp, err := (&http.Client{Timeout: reqTimeout, Transport: x.transport}).Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var contentLength = cast.ToInt64(resp.Header.Get(headers.ContentLength))
	if contentLength > maxSize && len(size) < 2 { // 超限且没设置 maxSize
		return resp.Header, nil, errors.New(fmt.Sprintf("请求内容太大(%s)", resp.Header.Get(headers.ContentType)))
	}
	if contentLength > maxSize && len(size) >= 2 {
		// 超限切设置 maxSize，则返回 maxSize
		contentLength = maxSize
	}
	if contentLength == 0 {
		// 竟然有服务器不返回 ContentLength
		contentLength = maxSize
	}
	if readSize == 0 {
		readSize = contentLength
	}

	b, err := io.ReadAll(io.LimitReader(resp.Body, readSize))
	if err != nil {
		return nil, nil, err
	}
	if resp.StatusCode != 200 {
		if len(resp.Status) > 0 {
			return resp.Header, b, errors.New(resp.Status)
		}
		return resp.Header, b, errors.New(fmt.Sprintf("上游服务器返回错误(%d)", resp.StatusCode))
	}

	return resp.Header, b, nil
}

func (x *HttpClient) PostResponse(requestUrl, rawBody string) (map[string][]string, []byte, error) {
	req, err := http.NewRequest("POST", requestUrl, strings.NewReader(rawBody))
	if err != nil {
		return nil, nil, err
	}
	x.addHeaderParams(req)

	x.InitClient()
	resp, err := (&http.Client{Timeout: reqTimeout, Transport: x.transport}).Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	b, err := x.decodeEncoding(resp)

	return resp.Header, b, err
}

func (x *HttpClient) Head(requestUrl string) (http.Header, error) {
	req, err := http.NewRequest("HEAD", requestUrl, nil)
	if err != nil {
		return nil, err
	}
	x.addHeaderParams(req)

	x.InitClient()
	resp, err := (&http.Client{Timeout: reqTimeout, Transport: x.transport}).Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	return resp.Header, nil
}

func (x *HttpClient) Clone() HttpClient {
	return HttpClient{headers: x.GetHeaders()}
}
