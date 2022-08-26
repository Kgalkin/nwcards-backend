package email

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"log"
	"net"
	"net/mail"
	"net/smtp"
	"nwcards-backend/src/go/props"
	"os"
	"path/filepath"
)

var from string
var pass string

func init() {
	props := props.Get()
	from = props["company.email"].(string)
	pass = props["email.password"].(string)
}

func Send(to string, subject string, body string) error {
	msg := "Subject: " + subject + "\n" +
		"MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n" +
		body

	return sendRow(to, []byte(msg))
}

func sendRow(to string, body []byte) error {
	err := smtp.SendMail("smtp.gmail.com:587",
		smtp.PlainAuth("", from, pass, "smtp.gmail.com"),
		from, []string{to}, body)

	if err != nil {
		log.Printf("smtp error: %s", err)
	}
	return err
}

func SendWithFile(to string, subject string, body string, path string) error {
	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf("From: %s\r\n", from))
	buf.WriteString(fmt.Sprintf("To: %s\r\n", to))
	buf.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))

	boundary := "my-boundary-1101101"
	buf.WriteString("MIME-Version: 1.0\r\n")
	buf.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=%s\n",
		boundary))

	buf.WriteString(fmt.Sprintf("\r\n--%s\r\n", boundary))
	buf.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
	buf.WriteString(fmt.Sprintf("\r\n%s", body))

	fileName := filepath.Base(path)

	buf.WriteString(fmt.Sprintf("\r\n--%s\r\n", boundary))
	buf.WriteString("Content-Type: text/plain; charset=\"utf-8\"\r\n")
	buf.WriteString("Content-Transfer-Encoding: base64\r\n")
	buf.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=%s\r\n", fileName))
	buf.WriteString(fmt.Sprintf("Content-ID: <%s>\r\n\r\n", fileName))

	data, er := os.ReadFile(path)
	if er != nil {
		log.Println(er)
		return er
	}

	b := make([]byte, base64.StdEncoding.EncodedLen(len(data)))
	base64.StdEncoding.Encode(b, data)
	buf.Write(b)
	buf.WriteString(fmt.Sprintf("\r\n--%s", boundary))

	buf.WriteString("--")

	return sendRow(to, buf.Bytes())
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
