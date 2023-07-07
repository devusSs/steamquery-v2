package utils

import (
	"bytes"
	"embed"
	"fmt"
	"net/mail"
	"path/filepath"
	"text/template"
	"time"

	"gopkg.in/gomail.v2"
)

//go:embed templates/base.html templates/styles.html templates/status.html
var templatesFS embed.FS

var (
	smtpFrom   string
	smtpTo     string
	mailServer *gomail.Dialer
)

type EmailData struct {
	Subject string
	Data    interface{}
}

func ValidateMail(emailAddr string) error {
	_, err := mail.ParseAddress(emailAddr)
	return err
}

func InitMail(host string, port int, user, pass, from, to string) {
	smtpFrom = from
	smtpTo = to

	mailServer = gomail.NewDialer(host, port, user, pass)
}

func SendMail(data *EmailData) error {
	var body bytes.Buffer

	tmpl, err := parseTemplateDir()
	if err != nil {
		return err
	}

	if err := tmpl.ExecuteTemplate(&body, "status.html", &data); err != nil {
		return err
	}

	m := gomail.NewMessage()

	m.SetHeader("From", smtpFrom)
	m.SetHeader("To", smtpTo)
	m.SetHeader("Subject", data.Subject)
	m.SetBody("text/html", body.String())

	return mailServer.DialAndSend(m)
}

func parseTemplateDir() (*template.Template, error) {
	tmpl := template.New("")

	templateFiles := []string{
		"templates/base.html",
		"templates/styles.html",
		"templates/status.html",
	}

	for _, file := range templateFiles {
		tmplContent, err := templatesFS.ReadFile(file)
		if err != nil {
			return nil, err
		}

		templateName := filepath.Base(file)
		tmpl = tmpl.New(templateName)
		tmpl = template.Must(tmpl.Parse(string(tmplContent)))
	}

	return tmpl, nil
}

func GeneratePriceDropWarning(priceDifference float64) string {
	return fmt.Sprintf(
		"Since your last steamquery-v2 run prices dropped a lot.<br>Drop value: %.2f€<br>Timestamp: %s",
		priceDifference,
		time.Now().Local().String(),
	)
}

func GenerateRunSummary(priceDifference float64) string {
	return fmt.Sprintf(
		"Your last steamquery-v2 run summary:<br>Price difference: %.2f€<br>Timestamp: %s",
		priceDifference, time.Now().Local().String())
}

func GenerateFailRunSummary(err error) string {
	return fmt.Sprintf(
		"Your last steamquery-v2 run failed.<br>Error: %s<br>Timestamp: %s",
		err.Error(),
		time.Now().Local().String(),
	)
}
