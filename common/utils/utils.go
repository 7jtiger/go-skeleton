package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"

	"gopkg.in/gomail.v2"
)

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
	prefix := prefixAdj[rand.Intn(len(prefixAdj))]
	main := mainAdj[rand.Intn(len(mainAdj))]
	noun := nouns[rand.Intn(len(nouns))]

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

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("cloudflare upload failed with status %d", resp.StatusCode)
		}
	}

	return &cldFlrInfos, nil
}

func GetFileType(filename string) int {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".jfif":
		return 0
	case ".mp4", ".mov", ".avi", ".mkv", ".mpg", ".mpeg", ".m4v", ".webm":
		return 1
	}
	return 2
}
