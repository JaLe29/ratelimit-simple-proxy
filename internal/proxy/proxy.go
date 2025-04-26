package proxy

import (
	"io"
	"net/http"
	"net/url"
)

var hostMap = map[string]string{
	"example.com":    "http://localhost:8081",
	"test.com":       "http://localhost:8082",
	"localhost:8080": "https://kamdoautoskoly.cz",
}

func ProxyHandler(w http.ResponseWriter, r *http.Request) {
	targetBase, ok := hostMap[r.Host]
	if !ok {
		http.Error(w, "Host '"+r.Host+"' not found", http.StatusBadGateway)
		return
	}

	targetURL, err := url.Parse(targetBase)
	if err != nil {
		http.Error(w, "Invalid target URL", http.StatusInternalServerError)
		return
	}

	proxyReq, err := http.NewRequest(r.Method, targetURL.String()+r.URL.RequestURI(), r.Body)
	if err != nil {
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}
	proxyReq.Header = r.Header.Clone()

	client := &http.Client{}
	resp, err := client.Do(proxyReq)

	if err != nil {
		http.Error(w, "Request failed", http.StatusBadGateway)
		return
	}

	defer resp.Body.Close()

	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}

	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}
