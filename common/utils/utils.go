package utils

import (
	"bufio"
	"context"
	"fmt"
	"net/smtp"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/jlaffaye/ftp"
	slack "github.com/m0t0k1ch1/go-slack-poster"
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

func SendGoMail(toUser, nick, body string) error {
	smtpHost := "smtp.gmail.com"
	// smtpPort := 587
	//smtpID := "customer@wemadetree.com.sg"
	smtpID := "7jtiger@gmail.com"
	smtpPW := "sqwer" //setting app password

	sender := gomail.NewDialer(smtpHost, 587, smtpID, smtpPW)
	mail, err := sender.Dial()
	if err != nil {
		return err
	}

	msg := gomail.NewMessage()
	// msg.SetHeader("From", smtpID)
	msg.SetHeader("From", "develop@gmail.com")
	msg.SetAddressHeader("To", toUser, nick)
	msg.SetHeader("Subject", "test mail")
	msg.SetBody("text/html", body)

	if err := gomail.Send(mail, msg); err != nil {
		return err
	}

	msg.Reset()
	return nil

	/*
		smtpHost := "smtp.gmail.com"
		// smtpPort := 587
		smtpID := "admin@com.io"
		smtpPW := "ds" //setting app password

		m := gomail.NewMessage()
		m.SetHeader("From", "admin@com.io")
		m.SetHeader("To", toUser)
		// m.SetAddressHeader("Cc", "dan@example.com", "Dan")
		m.SetHeader("Subject", "Test!")
		m.SetBody("text/html", body)
		// m.Attach("/home/Alex/lolcat.jpg")

		d := gomail.NewDialer(smtpHost, 587, smtpID, smtpPW)

		if err := d.DialAndSend(m); err != nil {
			return err
		}

		return nil
	*/

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
