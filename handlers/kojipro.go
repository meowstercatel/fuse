package handlers

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"
)

var kojiProUrl = "https://mgstpp-game.konamionline.com/"

const browserUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/148.0.0.0 Safari/537.36"

func ToKojiPro(URL string, body io.Reader, length int64) (*http.Response, error) {
	c := http.Client{Timeout: time.Second * 10, Transport: &http.Transport{
		DisableCompression: true,
	}}

	newUrl, err := url.JoinPath(kojiProUrl, URL)
	if err != nil {
		return nil, fmt.Errorf("cannot make an url from %s: %w", URL, err)
	}

	req, err := http.NewRequest(http.MethodPost, newUrl, body)
	if err != nil {
		return nil, fmt.Errorf("cannot create request to kojipro: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br, zstd")
	req.Header.Set("Connection", "Keep-Alive")

	req.Header.Set("User-Agent", browserUserAgent)
	req.Header.Set("Sec-CH-UA", `"Chromium";v="148", "Google Chrome";v="148", "Not/A)Brand";v="99"`)
	req.Header.Set("Sec-CH-UA-Mobile", "?0")
	req.Header.Set("Sec-CH-UA-Platform", `"Windows"`)
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	req.Header.Del("Transfer-Encoding")

	req.ContentLength = length

	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cannot make a request to %s: %w", newUrl, err)
	}

	slog.Debug("resp", "code", resp.StatusCode, "content-length", resp.ContentLength, "transfer-encoding", resp.TransferEncoding, "uncompressed", resp.Uncompressed)
	headers := ""
	for k, v := range resp.Header {
		headers += fmt.Sprintf("%s: %s ::: ", k, v)
	}
	slog.Debug("headers", "values", headers)

	return resp, nil
}
