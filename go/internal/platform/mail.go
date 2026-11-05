package platform

import (
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"strconv"
	"strings"
	"time"
)

// SMTPConfig bülten gönderimi için SMTP yapılandırması.
type SMTPConfig struct {
	Host        string
	Port        int
	User        string
	Pass        string
	From        string
	FromName    string
	ImplicitTLS bool // 465 (SMTPS)
}

// Configured yapılandırma yeterli mi (host + from dolu)?
func (c SMTPConfig) Configured() bool {
	return strings.TrimSpace(c.Host) != "" && strings.TrimSpace(c.From) != ""
}

// SendMail tek bir alıcıya HTML e-posta gönderir (UTF-8, MIME/base64).
// 465 için implicit TLS, aksi halde düz/STARTTLS (net/smtp otomatik yükseltir).
func (c SMTPConfig) SendMail(to, subject, htmlBody string) error {
	if !c.Configured() {
		return errors.New("SMTP yapılandırılmamış")
	}
	if strings.TrimSpace(to) == "" {
		return errors.New("alıcı boş")
	}
	return c.deliver(to, buildMIME(c, to, subject, htmlBody))
}

// SendMailAttach HTML e-postayı tek ek (ör. gazete PDF) ile gönderir.
func (c SMTPConfig) SendMailAttach(to, subject, htmlBody, attachName string, attachData []byte) error {
	if !c.Configured() {
		return errors.New("SMTP yapılandırılmamış")
	}
	if strings.TrimSpace(to) == "" {
		return errors.New("alıcı boş")
	}
	return c.deliver(to, buildMIMEAttach(c, to, subject, htmlBody, attachName, attachData))
}

// deliver MIME mesajını SMTP üzerinden gönderir.
func (c SMTPConfig) deliver(to string, msg []byte) error {
	addr := net.JoinHostPort(c.Host, strconv.Itoa(c.Port))

	var client *smtp.Client
	if c.ImplicitTLS {
		conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: c.Host, MinVersion: tls.VersionTLS12})
		if err != nil {
			return err
		}
		client, err = smtp.NewClient(conn, c.Host)
		if err != nil {
			return err
		}
	} else {
		var err error
		client, err = smtp.Dial(addr)
		if err != nil {
			return err
		}
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(&tls.Config{ServerName: c.Host, MinVersion: tls.VersionTLS12}); err != nil {
				return err
			}
		}
	}
	defer client.Close()

	if c.User != "" {
		auth := smtp.PlainAuth("", c.User, c.Pass, c.Host)
		if ok, _ := client.Extension("AUTH"); ok {
			if err := client.Auth(auth); err != nil {
				return err
			}
		}
	}
	from := c.From
	if err := client.Mail(from); err != nil {
		return err
	}
	if err := client.Rcpt(to); err != nil {
		return err
	}
	wc, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := wc.Write(msg); err != nil {
		return err
	}
	if err := wc.Close(); err != nil {
		return err
	}
	return client.Quit()
}

// buildMIMEAttach HTML gövde + tek ek (base64, multipart/mixed) üretir.
func buildMIMEAttach(c SMTPConfig, to, subject, htmlBody, attachName string, attachData []byte) []byte {
	fromHdr := c.From
	if c.FromName != "" {
		fromHdr = fmt.Sprintf("%s <%s>", c.FromName, c.From)
	}
	boundary := "tb-gazete-" + encodeWord(fmt.Sprintf("%d", time.Now().UnixNano()))
	if attachName == "" {
		attachName = "ek.bin"
	}
	var b strings.Builder
	b.WriteString("From: " + fromHdr + "\r\n")
	b.WriteString("To: " + to + "\r\n")
	b.WriteString("Subject: " + encodeWord(subject) + "\r\n")
	b.WriteString("Date: " + time.Now().Format(time.RFC1123Z) + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: multipart/mixed; boundary=\"" + boundary + "\"\r\n\r\n")
	b.WriteString("--" + boundary + "\r\n")
	b.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	b.WriteString("Content-Transfer-Encoding: base64\r\n\r\n")
	enc := base64.StdEncoding.EncodeToString([]byte(htmlBody))
	for i := 0; i < len(enc); i += 76 {
		end := i + 76
		if end > len(enc) {
			end = len(enc)
		}
		b.WriteString(enc[i:end] + "\r\n")
	}
	b.WriteString("--" + boundary + "\r\n")
	b.WriteString("Content-Type: application/pdf; name=\"" + attachName + "\"\r\n")
	b.WriteString("Content-Transfer-Encoding: base64\r\n")
	b.WriteString("Content-Disposition: attachment; filename=\"" + attachName + "\"\r\n\r\n")
	enc = base64.StdEncoding.EncodeToString(attachData)
	for i := 0; i < len(enc); i += 76 {
		end := i + 76
		if end > len(enc) {
			end = len(enc)
		}
		b.WriteString(enc[i:end] + "\r\n")
	}
	b.WriteString("--" + boundary + "--\r\n")
	return []byte(b.String())
}

func buildMIME(c SMTPConfig, to, subject, htmlBody string) []byte {
	fromHdr := c.From
	if c.FromName != "" {
		fromHdr = fmt.Sprintf("%s <%s>", c.FromName, c.From)
	}
	var b strings.Builder
	b.WriteString("From: " + fromHdr + "\r\n")
	b.WriteString("To: " + to + "\r\n")
	b.WriteString("Subject: " + encodeWord(subject) + "\r\n")
	b.WriteString("Date: " + time.Now().Format(time.RFC1123Z) + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	b.WriteString("Content-Transfer-Encoding: base64\r\n")
	b.WriteString("\r\n")
	enc := base64.StdEncoding.EncodeToString([]byte(htmlBody))
	for i := 0; i < len(enc); i += 76 {
		end := i + 76
		if end > len(enc) {
			end = len(enc)
		}
		b.WriteString(enc[i:end] + "\r\n")
	}
	return []byte(b.String())
}

// encodeWord başlığı RFC 2047 UTF-8 base64 olarak kodlar.
func encodeWord(s string) string {
	return "=?UTF-8?B?" + base64.StdEncoding.EncodeToString([]byte(s)) + "?="
}
