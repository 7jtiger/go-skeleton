package controller

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"testing"
)

func Get(host, relativePath string, keys []string, values []string) (string, error) {
	// if len(keys) != len(values) {
	// 	return "", fmt.Errorf("mismatch length of keys and values")
	// }
	u := url.URL{Scheme: "http", Host: host, Path: relativePath}
	if request, err := http.NewRequest("GET", u.String(), nil); err != nil {
		return "", err
	} else {
		q := request.URL.Query()
		for i, key := range keys {
			q.Add(key, values[i])
		}
		request.URL.RawQuery = q.Encode()

		client := http.Client{
			//Timeout: 50e9,
		}

		if response, err := client.Do(request); err != nil {
			return "", err
		} else {
			defer response.Body.Close()

			if body, err := io.ReadAll(response.Body); err != nil {
				return "", err
			} else {
				return string(body), nil
			}
		}
	}
}

func TestGetItem(t *testing.T) {
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	// qurl := "api/v3/ticker/bookTicker"
	qurl := "/api/v3/ticker/price"

	// target := fmt.Sprintf("%d-%02d-%02d", 2020, 12, 23)
	var key = []string{"symbol"}
	var value = []string{"TRXUSDT"}

	// param := "daylog/Chance/Event/" + target
	// res, _ := util.Get(*targetUrl, qurl+param, key, value)
	res, _ := Get(*targetUrl, qurl, key, value)

	fmt.Println(res)
}
