package main

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"os"
	"strings"

	"github.com/Crapworks/smtp2slack/auth"
	"github.com/Crapworks/smtp2slack/kubernetes"
	"github.com/ProtonMail/gopenpgp/v2/helper"
	"github.com/alexflint/go-arg"
	"github.com/mhale/smtpd"
	"github.com/nlopes/slack"
	"github.com/veqryn/go-email/email"
)

type SmtpToSlack struct {
	slack      *slack.Client
	authToken  string
	channels   []string
	encSenders []string
	auth       auth.SmtpAuth
	pubKey     string
	TLSCert    string
	TLSKey     string
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
		log.Printf("error uploading text/plain message: %s", err)
		return err
	}
	err = s.uploadSlack(from, to[0], msg, "text/html")
	if err != nil {
		log.Printf("error uploading text/html message: %s", err)
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

	if s.TLSCert != "" {
		srv.TLSRequired = true
		srv.TLSListener = false // Use STARTTLS instead
		if err := srv.ConfigureTLS(s.TLSCert, s.TLSKey); err != nil {
			return err
		}
	}

	return srv.ListenAndServe()
}

func parseWatchSecret(watchsecret string) (string, string, error) {
	var ParseError = fmt.Errorf("unable to parse watchsecret string: %s. format: namespace:secretname", watchsecret)
	s := strings.Split(watchsecret, ":")
	if len(s) != 2 {
		return "", "", ParseError
	}

	namespace, secret := s[0], s[1]

	if namespace == "" || secret == "" {
		return "", "", ParseError
	}
	return namespace, secret, nil
}

func main() {
	var args struct {
		Addr             string   `arg:"env:LISTEN_ADDR" default:"0.0.0.0:2525" help:"address string to listen on"`
		Channel          []string `arg:"env:CHANNEL,required" help:"channel to forward the mails to"`
		Token            string   `arg:"env:TOKEN,required" help:"slack authentication token"`
		Auth             string   `arg:"env:AUTH" help:"user:passwd combination for authentication"`
		EncryptedSenders []string `arg:"env:ENCRYPTED_SENDERS" help:"sender addresses which mails should be encrypted"`
		PubKey           string   `arg:"env:PUBKEY" help:"path to a file that contains the public key for encryption"`
		TLSCert          string   `arg:"env:TLSCERT" help:"path to tls certificate"`
		TLSKey           string   `arg:"env:TLSKEY" help:"path to tls key"`
		WatchSecret      string   `arg:"env:WATCHSECRET" help:"watch secret and restart if it is changed. format is namespace:secretname"`
	}
	arg.MustParse(&args)

	if len(args.EncryptedSenders) > 0 && args.PubKey == "" {
		log.Fatal("-encryptedsenders specified but no -pubkey provided for encryption")
	}

	if (args.TLSCert != "" && args.TLSKey == "") || (args.TLSKey != "" && args.TLSCert == "") {
		log.Fatal("--tlskey and --tlscert are required to be set together but only one is set")
	}

	if args.WatchSecret != "" {
		namespace, secret, err := parseWatchSecret(args.WatchSecret)
		if err != nil {
			log.Printf("[watch] error: %s", err)
		} else {
			log.Printf("[watch] watching secret %s in namespace %s for changes.", secret, namespace)
			k8s, err := kubernetes.New()
			if err != nil {
				log.Printf("failed to create kubernetes client: %s", err)
			} else {
				go k8s.WatchSecret(namespace, secret)
			}
		}
	}

	var publicKey []byte
	var err error
	if args.PubKey != "" {
		publicKey, err = os.ReadFile(args.PubKey)
		if err != nil {
			log.Fatalf("error opening public key file: %s", err)
		}
	}

	auth := auth.New()
	err = auth.SetSMTPAuth(args.Auth)
	if err != nil {
		log.Printf("unable to parse credentials: %s", err)
	}

	s := SmtpToSlack{
		slack:      slack.New(args.Token),
		authToken:  args.Token,
		channels:   args.Channel,
		auth:       *auth,
		encSenders: args.EncryptedSenders,
		pubKey:     string(publicKey),
		TLSCert:    args.TLSCert,
		TLSKey:     args.TLSKey,
	}

	log.Printf("[server] listening for mail on %s", args.Addr)
	err = s.listenAndServe(args.Addr)
	if err != nil {
		log.Printf("error starting server: %s", err)
	}
}
