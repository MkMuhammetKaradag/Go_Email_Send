package main

import (
	"bytes"
	"html/template"
	"log"
	"net/smtp"
	"os"

	"github.com/joho/godotenv"
)

// EmailData şablon için dinamik verileri temsil eder
type EmailData struct {
	ActivationCode string
	ActivationLink string
}

// Şablon oluşturma fonksiyonu
func renderTemplate(templatePath string, data EmailData) (string, error) {
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// E-posta gönderme fonksiyonu
func sendEmail(subject, body, recipient string) error {
	// Çevresel değişkenleri yükle
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Çevresel değişkenler yüklenemedi:", err)
	}

	// .env dosyasından SMTP bilgilerini al
	from := os.Getenv("SMTP_EMAIL")
	password := os.Getenv("SMTP_PASSWORD")
	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")


	auth := smtp.PlainAuth("", from, password, smtpHost)

	msg := []byte("Subject: " + subject + "\r\n" +
		"Content-Type: text/html; charset=\"utf-8\"\r\n" +
		"\r\n" +
		body)

		err = smtp.SendMail(smtpHost+":"+smtpPort, auth, from, []string{recipient}, msg)
		if err != nil {
			return err
		}
		return nil
}

func main() {
	// Aktivasyon e-postası için dinamik veriler
	data := EmailData{
		ActivationCode: "123456",
		ActivationLink: "https://example.com/activate?code=123456",
	}

	// Şablonu oluştur
	body, err := renderTemplate("templates/activation_email.html", data)
	if err != nil {
		log.Fatal("Şablon oluşturulamadı:", err)
	}

	// E-posta gönder
	err = sendEmail("Hesap Aktivasyonu", body, "recipient@example.com")
	if err != nil {
		log.Fatal("E-posta gönderilemedi:", err)
	}

	log.Println("E-posta başarıyla gönderildi.")
}
