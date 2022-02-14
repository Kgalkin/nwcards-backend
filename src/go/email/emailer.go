package email

import (
	"NorthwindREST/src/go/props"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/mail"
	"net/smtp"
)

var from string
var pass string

func init() {
	props := props.Get()
	from = props["email.login"]
	pass = props["email.password"]
}

func Send(to string, subject string, body string) error {
	msg := "Subject: " + subject + "\n" +
		"MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n" +
		body

	err := smtp.SendMail("smtp.gmail.com:587",
		smtp.PlainAuth("", from, pass, "smtp.gmail.com"),
		from, []string{to}, []byte(msg))

	if err != nil {
		log.Printf("smtp error: %s", err)
	}
	return err
}

func SendEmail(recipient, text string) error {
	from := mail.Address{"", "galkin_kirill@mail.ru"}
	to := mail.Address{"", recipient}
	subj := "This is the email subject"
	body := "This is an example body.\n" + text

	// Setup headers
	headers := make(map[string]string)
	headers["From"] = from.String()
	headers["To"] = to.String()
	headers["Subject"] = subj

	// Setup message
	message := ""
	for k, v := range headers {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + body

	// Connect to the SMTP Server
	servername := "smtp.mail.ru:465"

	host, _, _ := net.SplitHostPort(servername)

	auth := smtp.PlainAuth("", "galkin_kirill@mail.ru", "stop", host)

	// TLS config
	tlsconfig := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         host,
	}

	// Here is the key, you need to call tls.Dial instead of smtp.Dial
	// for smtp servers running on 465 that require an ssl connection
	// from the very beginning (no starttls)
	conn, err := tls.Dial("tcp", servername, tlsconfig)
	if err != nil {
		log.Panic(err)
	}

	c, err := smtp.NewClient(conn, host)
	if err != nil {
		log.Panic(err)
	}

	defer c.Quit()

	// Auth
	if err = c.Auth(auth); err != nil {
		log.Panic(err)
	}

	// To && From
	if err = c.Mail(from.Address); err != nil {
		log.Panic(err)
	}

	if err = c.Rcpt(to.Address); err != nil {
		log.Panic(err)
	}

	// Data
	w, err := c.Data()
	if err != nil {
		log.Panic(err)
	}

	_, err = w.Write([]byte(message))
	if err != nil {
		log.Panic(err)
	}

	err = w.Close()
	if err != nil {
		log.Panic(err)
	}

	return nil
}
