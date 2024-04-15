package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/Crapworks/smtp2slack/auth"
	"github.com/ProtonMail/gopenpgp/v2/helper"
	"github.com/mhale/smtpd"
	"github.com/nlopes/slack"
	"github.com/veqryn/go-email/email"
)

type Config struct {
	addr       string
	authToken  string
	channels   []string
	encSenders []string
	pubKey     string
	auth       auth.SmtpAuth
}

type SmtpToSlack struct {
	slack      *slack.Client
	authToken  string
	channels   []string
	encSenders []string
	auth       auth.SmtpAuth
	pubKey     string
}

func New(cfg *Config) *SmtpToSlack {
	return &SmtpToSlack{
		slack:      slack.New(cfg.authToken),
		authToken:  cfg.authToken,
		channels:   cfg.channels,
		encSenders: cfg.encSenders,
		auth:       cfg.auth,
		pubKey:     cfg.pubKey,
	}
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func (s *SmtpToSlack) uploadSlack(sender string, recipient string, msg *email.Message, mimetype string) error {
	subject := msg.Header.Get("Subject")
	fromHeader := msg.Header.Get("From")

	var filetype string
	if mimetype == "text/plain" {
		filetype = "text"
	}
	if mimetype == "text/html" {
		filetype = "html"
	}

	for _, part := range msg.MessagesContentTypePrefix(mimetype) {
		content := string(part.Body)
		if stringInSlice(sender, s.encSenders) {
			encContent, err := helper.EncryptMessageArmored(s.pubKey, content)
			if err != nil {
				return err
			}
			content = string(encContent)
		}
		uploadparams := slack.FileUploadParameters{
			Channels:       s.channels,
			Title:          fmt.Sprintf("Subject: %s", subject),
			Filetype:       filetype,
			Content:        content,
			InitialComment: fmt.Sprintf("To: %s\nFrom: %s", recipient, fromHeader),
		}
		file, err := s.slack.UploadFile(uploadparams)
		if err != nil {
			return err
		}
		log.Printf("[upload] successfully sent to channel %s as %s file %s", s.channels, filetype, file.Name)
	}
	return nil
}

func (s *SmtpToSlack) mailHandler(origin net.Addr, from string, to []string, data []byte) error {
	log.Printf("[handler] handling message from %s to %s", from, to[0])
	msg, err := email.ParseMessage(bytes.NewReader(data))
	if err != nil {
		log.Printf("error parsing message: %s", err)
		return err
	}

	err = s.uploadSlack(from, to[0], msg, "text/plain")
	if err != nil {
		log.Printf("error parsing message: %s", err)
		return err
	}
	err = s.uploadSlack(from, to[0], msg, "text/html")
	if err != nil {
		log.Printf("error parsing message: %s", err)
		return err
	}
	return nil
}

func (s *SmtpToSlack) authHandler(remoteAddr net.Addr, mechanism string, username []byte, password []byte, _ []byte) (bool, error) {
	allow := s.auth.Match(string(username), string(password))
	log.Printf("[login] mechanism: %s user: %s from %s - successful: %t", mechanism, string(username), remoteAddr, allow)
	return allow, nil
}

func (s *SmtpToSlack) listenAndServe(addr string) error {
	srv := &smtpd.Server{
		Addr:         addr,
		Handler:      s.mailHandler,
		Appname:      "Smtp2Slack",
		Hostname:     "",
		AuthHandler:  nil,
		AuthRequired: false,
	}

	if s.auth.HasCredentials() {
		srv.AuthMechs = map[string]bool{"CRAM-MD5": false, "PLAIN": true, "LOGIN": true}
		srv.AuthHandler = s.authHandler
		srv.AuthRequired = true
	}

	return srv.ListenAndServe()
}

func main() {
	addStr := flag.String("addr", "0.0.0.0:2525", "address string to listen on")
	channel := flag.String("channel", "#mail", "channel to forward the mails to")
	token := flag.String("token", "", "slack authentication token")
	authStr := flag.String("auth", "", "user:passwd combination for authentication")
	encSenders := flag.String("encrypted_senders", "", "sender addresses which mails should be encrypted")
	pubKey := flag.String("pubkey", "", "path to a file that contains the public key for encryption")
	flag.Parse()

	// Validation
	if *token == "" {
		log.Fatal("-token parameter is missing but required")
	}

	if *encSenders != "" && *pubKey == "" {
		log.Fatal("-encrypted_senders specified but no -pubkey provided for encryption")
	}

	var publicKey []byte
	var err error
	if *pubKey != "" {
		publicKey, err = os.ReadFile(*pubKey)
		if err != nil {
			log.Fatalf("error opening public key file: %s", err)
		}
	}

	auth := auth.New()
	err = auth.SetSMTPAuth(*authStr)
	if err != nil {
		log.Printf("unable to parse credentials: %s", err)
	}
	s := New(&Config{
		addr:       *addStr,
		authToken:  *token,
		channels:   []string{*channel},
		auth:       *auth,
		encSenders: []string{*encSenders},
		pubKey:     string(publicKey),
	})

	log.Printf("[server] listening for mail on %s", *addStr)
	err = s.listenAndServe(*addStr)
	if err != nil {
		log.Printf("error starting server: %s", err)
	}
}
