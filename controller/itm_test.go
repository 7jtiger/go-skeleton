package controller

import (
	"flag"
	"fmt"
	"testing"
	// "gocv.io/x/gocv"
)

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
