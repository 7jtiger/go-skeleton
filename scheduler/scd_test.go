package scheduler

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"testing"
	"time"
	// log "banker/common/logger"
)

func TestSendMail(t *testing.T) {
	fmt.Println("Send mail")

	// SMTP server configuration.
	smtpServer := "smtp.mailplug.co.kr"
	auth := smtp.PlainAuth("", "jtiger@gurufin.com", "wlsh4156!", smtpServer)

	// Email details.
	from := "jtiger@gurufin.com"
	to := []string{"jtiger@groufin.com"}
	subject := "Subject: Test Mail\n"
	body := "This is a test email."

	// Construct the email message.
	msg := []byte(subject + "\n" + body)

	// TLS configuration
	tlsconfig := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         smtpServer,
	}

	// Connect to the SMTP server
	conn, err := tls.Dial("tcp", smtpServer+":465", tlsconfig)
	if err != nil {
		fmt.Println(fmt.Sprintf("Failed to connect to SMTP server: %v", err))
		return
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, smtpServer)
	if err != nil {
		fmt.Println(fmt.Sprintf("Failed to create SMTP client: %v", err))
		return
	}

	// Authenticate
	if err = client.Auth(auth); err != nil {
		fmt.Println(fmt.Sprintf("Failed to authenticate: %v", err))
		return
	}

	// Set the sender and recipient
	if err = client.Mail(from); err != nil {
		fmt.Println(fmt.Sprintf("Failed to set sender: %v", err))
		return
	}
	for _, addr := range to {
		if err = client.Rcpt(addr); err != nil {
			fmt.Println(fmt.Sprintf("Failed to set recipient: %v", err))
			return
		}
	}

	// Send the email body
	w, err := client.Data()
	if err != nil {
		fmt.Println(fmt.Sprintf("Failed to send data: %v", err))
		return
	}
	_, err = w.Write(msg)
	if err != nil {
		fmt.Println(fmt.Sprintf("Failed to write message: %v", err))
		return
	}
	err = w.Close()
	if err != nil {
		fmt.Println(fmt.Sprintf("Failed to close writer: %v", err))
		return
	}

	// Close the connection
	client.Quit()

	fmt.Println("Email sent successfully")
}

func TestDuration(t *testing.T) {
	var rettime time.Duration
	start := 1
	// duration := 86400

	tt := time.Now()
	if start == 0 { // 1day 00:00:00 start
		n := time.Date(tt.Year(), tt.Month(), tt.Day()+1, 0, 0, 0, 0, tt.Location())
		rettime = n.Sub(tt)
	} else if start == 1 { //**:**:00 start
		n := time.Date(tt.Year(), tt.Month(), tt.Day(), tt.Hour(), tt.Minute()+1, 0, 0, tt.Location())
		rettime = n.Sub(tt)
	} else if start == 5 { //duration 5min
		g := 5 - (tt.Minute() % int(5))
		if g == 0 {
			g = 5
		}
		n := time.Date(tt.Year(), tt.Month(), tt.Day(), tt.Hour(), tt.Minute()+g, 0, 0, tt.Location())
		rettime = n.Sub(tt)
	} else { // + start secont start
		n := time.Date(tt.Year(), tt.Month(), tt.Day(), tt.Hour(), tt.Minute(), tt.Second()+start, 0, tt.Location())
		rettime = n.Sub(tt)
	}

	fmt.Println(rettime)
}

func TestFstCheck(t *testing.T) {
	// s := &Schedule{
	// 	pdb: &models.BankerDB{},
	// }

	// deposits, err := s.pdb.GetDepositAll(1, protocol.Pagination{CurrentPage: 1, LastPage: 0, PagePerCount: 99})
	// if err != nil {
	// 	t.Errorf("Error getting deposits: %v", err)
	// 	return
	// }

	// for _, dep := range deposits {
	// 	t.Logf("Deposit: %v", dep)
	// }

}
