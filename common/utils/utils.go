package utils

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"mime/multipart"
	"net/http"
	"net/smtp"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/jlaffaye/ftp"
	slack "github.com/m0t0k1ch1/go-slack-poster"

	// "github.com/rs/zerolog/log"
	"gopkg.in/gomail.v2"
)

func HomeDir() string {
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	if usr, err := user.Current(); err == nil {
		return usr.HomeDir
	}
	return ""
}

func WorkingDir() string {
	workdir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		panic(err)
	} else {
		if strings.Contains(workdir, "/var/folders/") == true {
			workdir = "./"
		}
	}
	return workdir
}

func Trace() string {
	pc := make([]uintptr, 10)
	runtime.Callers(2, pc)
	f := runtime.FuncForPC(pc[0])
	path, line := f.FileLine(pc[0])
	function := func() string {
		splited := strings.Split(path, "/")
		if len(splited) > 0 {
			return splited[len(splited)-1]
		} else {
			return path
		}
	}()
	return fmt.Sprintf("path :%v, file: %v, func: %v, line: %v", path, function, f.Name(), line)
}

func Trace3() string {
	pc := make([]uintptr, 10)
	runtime.Callers(3, pc)
	f := runtime.FuncForPC(pc[0])
	path, line := f.FileLine(pc[0])
	function := func() string {
		splited := strings.Split(path, "/")
		if len(splited) > 0 {
			return splited[len(splited)-1]
		} else {
			return path
		}
	}()
	return fmt.Sprintf("path :%v, file: %v, func: %v, line: %v", path, function, f.Name(), line)
}

func SendMail(toUser, subj, body string) error {
	auth := smtp.PlainAuth("", "sender@live.com", "pwd", "smtp.live.com")

	from := "sender@live.com"
	// to := []string{"receiver@live.com"} // 복수 수신자 가능
	to := []string{toUser}

	// 메시지 작성
	headerSubject := "Subject: " + subj + "\r\n"
	headerBlank := "\r\n"
	mbody := "test mail body blabla" + body + "\r\n"

	msg := []byte(headerSubject + headerBlank + mbody)

	// 메일 보내기
	err := smtp.SendMail("smtp.live.com:587", auth, from, to, msg)
	if err != nil {
		return err
	}

	return nil
}

func SendGoMail(toUser, nick, subject, body string) error {
	smtpHost := "smtp.gmail.com"
	// smtpPort := 587
	smtpID := "livein.devx@gmail.com"
	// smtpPW := "flqldlsepqmx12#$" //setting app password
	smtpPW := "rdgntnbajsstjurm" //setting app password
	/*
		smtpID := os.Getenv("SMTP_EMAIL")
		if smtpID == "" {
			smtpID = "livein.devx@gmail.com" // fallback
		}
		smtpPW := os.Getenv("SMTP_APP_PASSWORD")
		if smtpPW == "" {
			smtpPW = "flqmdls12#$" // fallback - Google 앱 비밀번호로 교체 필요
		}
	*/

	sender := gomail.NewDialer(smtpHost, 587, smtpID, smtpPW)
	mail, err := sender.Dial()
	if err != nil {
		return err
	}

	msg := gomail.NewMessage()
	// msg.SetHeader("From", "develop@gmail.com")
	msg.SetHeader("From", "livein.devx@gmail.com")
	msg.SetAddressHeader("To", toUser, nick)
	msg.SetHeader("Subject", subject)
	msg.SetBody("text/html", body)

	if err := gomail.Send(mail, msg); err != nil {
		return err
	}

	msg.Reset()
	return nil
}

func Mkdirp(path string) bool {

	if _, err := os.Stat(path); os.IsNotExist(err) {
		if merr := os.MkdirAll(path, 0777); merr != nil {
			fmt.Println(merr.Error())
			return false
		}
		//fmt.Println("----- ")
	}
	fmt.Println(path)
	return true
}

func UploadFtp(fPath string, destPath string) error {
	sHost := "host"
	sUser := "user"
	sPass := "pass"

	for i := 0; i < 3; i++ {
		//xcache.kinxcdn.com:21
		//sdfd / chedn(!!)
		cnt, err := ftp.Dial(sHost, ftp.DialWithTimeout(5*time.Second))
		if err != nil {
			return err
		}
		defer cnt.Quit()

		if err := cnt.Login(sUser, sPass); err != nil {
			return err
		}

		file, err := os.Open("./" + fPath)
		if err != nil {
			return err
		}

		reader := bufio.NewReader(file)
		// hm/noti, hm/faq
		save := fmt.Sprintf("hm/%s", destPath)

		if err := cnt.Stor(save, reader); err != nil {
			return err
		}
		break
	}

	return nil
}

func AlertSlack(msg string) {
	client := slack.NewClient("xoxp-672058")
	// if err := client.SendMessage(context.Background(), "#alert", msg, nil); err != nil {
	if err := client.SendMessage(context.Background(), "#tev", msg, nil); err != nil {
		fmt.Println("err")
	}
	return
}

func MemUsage() (alloc, total, sys, heap uint64, numGC uint32) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	alloc = m.Alloc / 1024 / 1024
	total = m.TotalAlloc / 1024 / 1024
	sys = m.Sys / 1024 / 1024
	heap = m.HeapAlloc / 1024 / 1024
	numGC = m.NumGC

	return
}

func MemUsageString() string {
	alloc, total, sys, heap, numGC := MemUsage()
	return fmt.Sprintf("Alloc: %v MiB, TotalAlloc: %v MiB, Sys: %v MiB, HeapAlloc: %v MiB, NumGC: %v", alloc, total, sys, heap, numGC)
}

var prefixAdj = []string{
	"행복한", "즐거운", "기쁜", "신나는", "따뜻한",
	"활기찬", "희망찬", "든든한", "씩씩한", "당당한",
	"용감한", "지혜로운", "재미있는", "유쾌한", "친절한",
	"다정한", "푸른", "하얀", "붉은", "노란",
	"착한", "밝은", "맑은", "고요한", "평화로운",
	"싱그러운", "포근한", "부드러운", "조용한", "활발한",
	"건강한", "튼튼한", "열정적인", "정직한", "순수한",
	"충실한", "믿음직한", "자유로운", "편안한", "단정한",
}

// 주요 형용사
var mainAdj = []string{
	"작은", "귀여운", "예쁜", "멋진", "깜찍",
	"아기", "꼬마", "청순한", "고운", "상큼",
	"달콤", "향긋", "반짝반짝", "아름다운", "사랑스러운",
	"신비", "은은", "화사", "빛난", "영롱",
}

// 명사
var nouns = []string{
	// 동물
	"토끼", "강아지", "고양이", "판다", "사슴",
	"기린", "코알라", "펭귄", "다람쥐", "햄스터",
	"병아리", "물개", "돌고래", "여우", "사자",
	"호랑이", "곰돌이", "양", "알파카", "친칠라",

	// 자연
	"하늘", "바다", "구름", "별", "햇님",
	"달님", "무지개", "숲", "나무", "꽃송이",
	"산", "강", "계곡", "초원", "들꽃",
	"단풍", "눈송이", "새싹", "나뭇잎", "꽃잎",

	// 과일/식물
	"사과", "딸기", "복숭아", "포도", "오렌지",
	"레몬", "자두", "메론", "수박", "바나나",
	"장미", "민들레", "진달래", "개나리", "벚꽃",
	"튤립", "해바라기", "국화", "제비꽃", "달리아",

	// 음식
	"마카롱", "쿠키", "케이크", "푸딩", "캔디",
	"초콜릿", "아이스크림", "젤리", "카스테라", "와플",

	// 추상적 개념
	"미소", "행복", "사랑", "꿈", "희망",
	"기쁨", "축복", "행운", "소망", "기적",

	// 날씨/계절
	"봄날", "여름", "가을", "겨울", "햇살",
	"바람", "단비", "새벽", "노을", "황혼",
}

func GenDefNick() string {
	prefix := prefixAdj[rand.Intn(len(prefixAdj))-1]
	main := mainAdj[rand.Intn(len(mainAdj))-1]
	noun := nouns[rand.Intn(len(nouns))-1]

	return prefix + " " + main + " " + noun
}

func GenerateOTP() string {
	otp := fmt.Sprintf("%06d", rand.Intn(1000000))
	return otp
}

func UploadCldFlr(files []*multipart.FileHeader, accountID, apiToken string) (*map[string]string, error) {
	cldFlrInfos := make(map[string]string, 4)
	for _, file := range files {
		// Open the file
		src, err := file.Open()
		if err != nil {
			return nil, err
		}
		defer src.Close()

		// Create multipart form data
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		// Add file to form
		part, err := writer.CreateFormFile("file", file.Filename)
		if err != nil {
			return nil, err
		}

		if _, err := io.Copy(part, src); err != nil {
			return nil, err
		}

		writer.Close()

		// Create request
		url := fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/images/v1", accountID)
		req, err := http.NewRequest("POST", url, body)
		if err != nil {
			return nil, err
		}

		// Set headers
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiToken))

		// Send request
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		// Read response body
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		// cldFlrInfo := CldFlrInfo{}
		// Parse response to extract success, filename, and variants
		var tempResponse struct {
			Success bool `json:"success"`
			Result  struct {
				Filename string   `json:"filename"`
				Variants []string `json:"variants"`
			} `json:"result"`
		}

		if err := json.Unmarshal(bodyBytes, &tempResponse); err != nil {
			return nil, err
		}

		cldFlrInfos[file.Filename] = tempResponse.Result.Variants[0]

		//{true {bc1.jpg [https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/6e8b29d4-4378-499a-7334-bd019dc25900/public]}}
		// 2026-01-26T22:33:00.075+0900	INFO	info	{"Info": "Response body: %s{\n  \"result\": {\n    \"id\": \"100320e3-fb16-490a-7b21-a931feec6400\",\n    \"filename\": \"bc1.jpg\",\n    \"uploaded\": \"2026-01-26T13:30:51.197Z\",\n    \"requireSignedURLs\": false,\n    \"variants\": [\n      \"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/100320e3-fb16-490a-7b21-a931feec6400/public\"\n    ]\n  },\n  \"success\": true,\n  \"errors\": [],\n  \"messages\": []\n}"}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("cloudflare upload failed with status %d", resp.StatusCode)
		}
	}

	return &cldFlrInfos, nil
}
