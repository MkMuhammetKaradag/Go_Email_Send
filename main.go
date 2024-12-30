package main

import (
	"encoding/json"
	"log"
	"os"

	"bytes"
	"html/template"
	"net/smtp"

	"github.com/joho/godotenv"
	"github.com/streadway/amqp"
)

// EmailData şablon için dinamik verileri temsil eder
type EmailData struct {
	ActivationCode string
	UserName string
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
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Çevresel değişkenler yüklenemedi:", err)
	}

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

// RabbitMQ'dan mesaj tüketme fonksiyonu
func consumeAuthQueue() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Çevresel değişkenler yüklenemedi:", err)
	}

	rabbitMQURL := os.Getenv("RABBITMQ_URL")
	queueName := os.Getenv("EMAIL_QUEUE_NAME")

	conn, err := amqp.Dial(rabbitMQURL)
	if err != nil {
		log.Fatalf("RabbitMQ bağlantısı kurulamadı: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Kanal açılamadı: %v", err)
	}
	defer ch.Close()

	q, err := ch.QueueDeclare(
		queueName, 
		true,   
		false,  
		false,  
		false,  
		nil,    
	)
	if err != nil {
		log.Fatalf("Kuyruk bildirimi başarısız: %v", err)
	}

	msgs, err := ch.Consume(
		q.Name, 
		"",     
		true,   
		false,  
		false,  
		false,  
		nil,    
	)
	if err != nil {
		log.Fatalf("Mesaj tüketimi başarısız: %v", err)
	}

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			// Gelen ham mesajı yazdır
			log.Printf("Ham Mesaj: %s", string(d.Body))

			// Gelen mesajı işle
			var message map[string]interface{}
			err := json.Unmarshal(d.Body, &message)
			if err != nil {
				log.Printf("JSON çözümleme hatası: %v", err)
				continue
			}

			// Gelen mesajın tüm detaylarını yazdır
			log.Printf("Mesaj İçeriği: %+v", message)

			// Pattern içindeki cmd'yi kontrol et
			patternMap, patternOk := message["pattern"].(map[string]interface{})
			if !patternOk {
				log.Printf("Geçersiz pattern formatı: %+v", message)
				continue
			}

			cmd, cmdOk := patternMap["cmd"].(string)
			if !cmdOk {
				log.Printf("Komut eksik veya geçersiz: %+v", message)
				continue
			}

			// Data'yı al
			data, dataOk := message["data"].(map[string]interface{})
			if !dataOk {
				log.Printf("Geçersiz data formatı: %+v", message)
				continue
			}

			// Email ve aktivasyon kodu çıkarma
			email, emailOk := data["email"].(string)
			activationCode, codeOk := data["activation_code"].(string)
			templateName, templateOk := data["template_name"].(string)
			userName, userNameOk := data["userName"].(string)

			if !emailOk || !codeOk || !templateOk ||!userNameOk{
				log.Printf("Eksik email, aktivasyon kodu veya şablon adı: %+v", data)
				continue
			}

			var subject string
			switch cmd {
			case "active_user":
				subject = "Hesap Aktivasyonu"
			case "forgot_password":
				subject = "Şifre Sıfırlama"
			default:
				log.Printf("Desteklenmeyen komut: %v", cmd)
				continue
			}


			// Aktivasyon e-postası için dinamik veriler
			emailData := EmailData{
				ActivationCode: activationCode,
				UserName: userName,
			}

			// Şablonu oluştur
			body, err := renderTemplate("templates/"+templateName, emailData)
			if err != nil {
				log.Printf("Şablon oluşturulamadı: %v", err)
				continue
			}


			// E-posta gönder
			err = sendEmail(subject, body, email)
			if err != nil {
				log.Printf("E-posta gönderilemedi: %v", err)
				continue
			}

			log.Printf("E-posta başarıyla gönderildi. Alıcı: %s", email)
		}
	}()

	log.Printf(" [*] RabbitMQ kuyruğunu dinlemeye başladı. Çıkış için CTRL+C kullanın")
	<-forever
}

func main() {
	consumeAuthQueue()
}