package util

import (
	"compress/flate"
	"compress/gzip"
	"crypto/tls"
	"github.com/andybalholm/brotli"
	"github.com/zc310/headers"
	"io"
	"net/http"
	"strings"
	"time"
)

type HttpClient struct {
	headers map[string]string
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

	var transport = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	resp, err := (&http.Client{Timeout: time.Second * 15, Transport: transport}).Do(req)
	if err != nil {
		return nil, err
	}

	return x.decodeEncoding(resp)
}

func (x *HttpClient) Post(requestUrl, rawBody string) ([]byte, error) {
	req, err := http.NewRequest("POST", requestUrl, strings.NewReader(rawBody))
	if err != nil {
		return nil, err
	}
	x.addHeaderParams(req)

	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return nil, err
	}

	return x.decodeEncoding(resp)
}

func (x *HttpClient) GetResponse(requestUrl string) (http.Header, []byte, error) {
	req, err := http.NewRequest("GET", requestUrl, nil)
	if err != nil {
		return nil, nil, err
	}
	x.addHeaderParams(req)

	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return nil, nil, err
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}
	return resp.Header, b, nil
}

func (x *HttpClient) PostResponse(requestUrl, rawBody string) (map[string][]string, []byte, error) {
	req, err := http.NewRequest("POST", requestUrl, strings.NewReader(rawBody))
	if err != nil {
		return nil, nil, err
	}
	x.addHeaderParams(req)

	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return nil, nil, err
	}

	b, err := x.decodeEncoding(resp)

	return resp.Header, b, err
}

func (x *HttpClient) Head(requestUrl string) (http.Header, error) {
	req, err := http.NewRequest("HEAD", requestUrl, nil)
	if err != nil {
		return nil, err
	}
	x.addHeaderParams(req)

	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return nil, err
	}

	return resp.Header, nil
}

func (x *HttpClient) Clone() HttpClient {
	return HttpClient{headers: x.GetHeaders()}
}
