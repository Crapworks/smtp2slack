// Package auth handles the web UI and SMTP authentication
package auth

import (
	"regexp"
	"strings"

	"github.com/tg123/go-htpasswd"
)

type SmtpAuth struct {
	SMTPCredentials *htpasswd.File
}

func New() *SmtpAuth {
	return &SmtpAuth{}
}

func (s *SmtpAuth) SetSMTPAuth(data string) error {
	var err error

	credentials := s.credentialsFromString(data)
	if len(credentials) == 0 {
		return nil
	}

	r := strings.NewReader(strings.Join(credentials, "\n"))

	s.SMTPCredentials, err = htpasswd.NewFromReader(r, htpasswd.DefaultSystems, nil)
	if err != nil {
		return err
	}

	return nil
}

func (s *SmtpAuth) Match(username string, password string) bool {
	return s.SMTPCredentials.Match(username, password)
}

func (s *SmtpAuth) HasCredentials() bool {
	return s.SMTPCredentials != nil
}

func (s *SmtpAuth) credentialsFromString(data string) []string {
	re := regexp.MustCompile(`\s+`)

	words := re.Split(data, -1)
	credentials := []string{}
	for _, w := range words {
		if w != "" {
			credentials = append(credentials, w)
		}
	}

	return credentials
}
