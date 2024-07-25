package utils

import (
	"bytes"
	"flag"
	"io"
	"os"

	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"

	"net/http"
	"net/url"

	"github.com/google/uuid"
)

func PostForm(host, relativePath string, reqform url.Values, resp interface{}) error {
	u := url.URL{Scheme: "http", Host: host, Path: relativePath}

	// if resp, err := http.PostForm(u.String(), reqform); err != nil {
	//if request, err := http.NewRequest("POST", u.String(), strings.NewReader(reqform.Encode())); err != nil {
	request, err := http.PostForm(u.String(), reqform)
	if err != nil {
		return fmt.Errorf(fmt.Sprintf("http.NewRequest %v", err))
	}

	defer request.Body.Close()
	respBody, err := io.ReadAll(request.Body)
	if err == nil {
		str := string(respBody)
		fmt.Println(str)
	} else if err := json.Unmarshal(respBody, resp); err != nil {
		log.Printf("response: %v %v %q", host, relativePath, respBody)
		return fmt.Errorf(fmt.Sprintf("json.Unmarshal %v", err))
	} else {
		return nil
	}

	return nil
}

func Post(host, relativePath string, req interface{}, resp interface{}) error {
	u := url.URL{Scheme: "http", Host: host, Path: relativePath}

	if requestBody, err := json.Marshal(req); err != nil {
		return fmt.Errorf(fmt.Sprintf("json.Marshal %v", err))
	} else if request, err := http.NewRequest("POST", u.String(), bytes.NewBuffer(requestBody)); err != nil {
		return fmt.Errorf(fmt.Sprintf("http.NewRequest %v", err))
	} else {
		request.Header.Set("Content-type", "application/json")
		request.Header.Set("Authorization", "Bearer ") //access token을 넣는다.

		client := http.Client{
			//Timeout: 50e9,
		}

		if response, err := client.Do(request); err != nil {
			return fmt.Errorf(fmt.Sprintf("client.Do %v", err))
		} else {
			defer response.Body.Close()

			if body, err := io.ReadAll(response.Body); err != nil {
				return fmt.Errorf(fmt.Sprintf("ioutil.ReadAll %v", err))
			} else if err := json.Unmarshal(body, resp); err != nil {
				log.Printf("response: %v %v %q", host, relativePath, body)
				return fmt.Errorf(fmt.Sprintf("json.Unmarshal %v", err))
			} else {
				return nil
			}
		}
	}
}

func PostWithHeader(url string, req interface{}, headerKeys []string, headerValues []string) (map[string]interface{}, error) {
	if requestBody, err := json.Marshal(req); err != nil {
		return nil, fmt.Errorf(fmt.Sprintf("json.Marshal %v", err))
	} else if request, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody)); err != nil {
		return nil, fmt.Errorf(fmt.Sprintf("http.NewRequest %v", err))
	} else {
		for i, key := range headerKeys {
			request.Header.Add(key, headerValues[i])
		}

		if response, err := http.DefaultClient.Do(request); err != nil {
			return nil, fmt.Errorf(fmt.Sprintf("client.Do %v", err))
		} else {
			defer response.Body.Close()

			var resp map[string]interface{}

			if body, err := io.ReadAll(response.Body); err != nil {
				return nil, fmt.Errorf(fmt.Sprintf("ioutil.ReadAll %v", err))
			} else if err := json.Unmarshal(body, &resp); err != nil {

				log.Printf("response: %v %q", url, body)
				return nil, fmt.Errorf(fmt.Sprintf("json.Unmarshal %v", err))
			} else {
				return resp, nil
			}

		}

	}
}

func PostHeaderStr(url, req string, headerKeys []string, headerValues []string) (map[string]interface{}, error) {
	/* if requestBody, err := json.Marshal(req); err != nil {
		return nil, fmt.Errorf(fmt.Sprintf("json.Marshal %v", err))
	} else  */
	if request, err := http.NewRequest("POST", url, bytes.NewReader([]byte(req))); err != nil {
		return nil, fmt.Errorf(fmt.Sprintf("http.NewRequest %v", err))
	} else {
		for i, key := range headerKeys {
			request.Header.Add(key, headerValues[i])
		}

		if response, err := http.DefaultClient.Do(request); err != nil {
			return nil, fmt.Errorf(fmt.Sprintf("client.Do %v", err))
		} else {
			defer response.Body.Close()

			var resp map[string]interface{}

			if body, err := io.ReadAll(response.Body); err != nil {
				return nil, fmt.Errorf(fmt.Sprintf("ioutil.ReadAll %v", err))
			} else if err := json.Unmarshal(body, &resp); err != nil {

				log.Printf("response: %v %q", url, body)
				return nil, fmt.Errorf(fmt.Sprintf("json.Unmarshal %v", err))
			} else {
				return resp, nil
			}

		}

	}
}

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

func GetWithHeader(url string, keys []string, values []string, headerKeys []string, headerValues []string) (string, error) {
	// if len(keys) != len(values) {
	// 	return "", fmt.Errorf("mismatch length of keys and values")
	// }
	if request, err := http.NewRequest("GET", url, nil); err != nil {
		return "", err
	} else {
		q := request.URL.Query()
		for i, key := range keys {
			q.Add(key, values[i])
		}
		request.URL.RawQuery = q.Encode()

		for i, key := range headerKeys {
			request.Header.Add(key, headerValues[i])
		}

		client := &http.Client{}

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

func GetCtxTest(surl string) bool {
	targetUrl := flag.String("target", surl, "target server url")
	qurl := "/bcx/v01/"

	target := fmt.Sprintf("%d-%02d-%02d", 2020, 12, 23)
	var key []string
	var value []string

	param := "daylog/Chance/Event/" + target
	res, _ := Get(*targetUrl, qurl+param, key, value)
	fmt.Println(res)

	return true
}

func PostTest(surl string) bool {
	targetUrl := flag.String("target", surl, "target server url")
	qurl := "/adm/forgo/"
	var param string = "test1"

	resp := map[string]interface{}{}

	if err := Post(*targetUrl, qurl+param, map[string]interface{}{
		"id": param,
	}, &resp); err != nil {
		log.Printf(qurl, err)
	} else {
		log.Printf(qurl, "response ok")
		return true
	}
	return true
}

func SendChatAlert(mod, body string) bool {
	pbytes, _ := json.Marshal(map[string]interface{}{"text": body})
	buff := bytes.NewBuffer(pbytes)
	if mod == "prod" {
		_, err := http.Post("https://chat.googleapis.com/v1/spaces/AAAA/messages?key=AIza%3D", "application/json", buff)
		if err != nil {
			return false
		}
	} else if mod == "dq" {
		_, err := http.Post("https://chat.googleapis.com/v1/spaces/AAAA8/messages?key=AIza&token=mq3D", "application/json", buff)
		if err != nil {
			return false
		}
	} else {
		_, err := http.Post("https://chat.googleapis.com/v1/spaces/AAAA/messages?key=AIzaxbadw%3D", "application/json", buff)
		if err != nil {
			return false
		}
	}
	//fmt.Println("resp ", resp)
	return true
}

func SendTelegramAlert(mod, body string) bool {
	path, _ := os.Getwd()
	var msg string
	if mod == "prod" {
		msg = "!!!From Prod-live stage!!! : \n" + body + "\nModule : " + path
	} else if mod == "beta" {
		msg = "From beta stage : \n" + body
	} else {
		msg = "Test Message : \n" + body
	}

	pbytes, _ := json.Marshal(map[string]interface{}{"chat_id": -1002091099927, "text": msg})
	buff := bytes.NewBuffer(pbytes)
	if _, err := http.Post("https://api.telegram.org/bot6488619941:AAHAG7mTP6fUoQljK_sGhyv44n-AmGv9Wxs/sendMessage", "application/json", buff); err != nil {
		return false
	}
	//fmt.Println("resp ", resp)
	return true
}

func GenUuid() string {
	uuid := uuid.New()
	uuid4 := hex.EncodeToString(uuid[:])
	return uuid4
}
