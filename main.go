package main

import (
	"fmt"
	"log"
	smtpv1 "github.com/Vyacheslav1557/ms-email/pkg/go/gen/smtp/v1"
	"net/smtp"

	"github.com/ilyakaznacheev/cleanenv"
	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/protobuf/proto"
)

type Config struct {
	Env      string `env:"ENV" required:"true"`
	Email    string `env:"EMAIL" required:"true"`
	Password string `env:"PASSWORD" required:"true"`
	RabbitDSN string `env:"RABBIT_DSN" required:"true"`
}

func SetupConfig() *Config {
	var cfg Config
	err := cleanenv.ReadConfig(".env", &cfg)
	if err != nil {
		failOnError(err, "error reading env")
	}
	return &cfg
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}

func main() {
	cfg := SetupConfig()

	to := []string{
		cfg.Email,
	}

	auth := smtp.PlainAuth("", cfg.Email, cfg.Password, "smtp.gmail.com")

	conn, err := amqp.Dial(cfg.RabbitDSN)
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	q, err := ch.QueueDeclare(
		"hello",
		false,
		false,
		false,
		false,
		nil,
	)

	failOnError(err, "Failed to declare a queue")

	msgs, err := ch.Consume(
		q.Name,
		"",
		true,
		false,
		false,
		false,
		nil,
	)

	failOnError(err, "Failed to register a consumer")

	var forever chan struct{}

	go func() {
		for d := range msgs {
			req := smtpv1.EmailRequest{}
			proto.Unmarshal(d.Body, &req)
			msg := fmt.Sprintf("From: %s\nTo: %s\nSubject: %s\n%s", cfg.Email, req.To, req.Subject, req.Body)
			err := smtp.SendMail("smtp.gmail.com:587", auth, cfg.Email, to, []byte(msg))
			if err != nil {
				failOnError(err, "cannot send email")
			}
		}
	}()

	log.Printf("[*] Waiting for messages.")
	<-forever
}
