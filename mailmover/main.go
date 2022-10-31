package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/textproto"
	"os"
	"regexp"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"
	"github.com/philandstuff/dhall-golang/v6"
)

type Config struct {
	IMAPHost     string
	SMTPHost     string
	Mailbox      string
	TrashMailbox string
	From         string
	FromName         string
	Login        struct {
		Username string
		Password string
	}
	Rules [](struct {
		From        string
		To          string
		RewriteFrom bool
	})
}

func readConfig(path string) (Config, error) {
	var config Config
	err := dhall.UnmarshalFile(path, &config)
	return config, err

}

func mailConnection(host, username, password string) (*client.Client, error) {
	c, err := client.DialTLS(host, nil)
	if err != nil {
		return nil, err
	}
	log.Println("Connected")

	// Login
	if err := c.Login(username, password); err != nil {
		return nil, err
	}
	log.Println("Logged in")
	return c, nil
}

func sendMail(config Config, from, subject, to,contenttype string, body []byte) error {
	auth := sasl.NewPlainClient("", config.Login.Username, config.Login.Password)
	msg := bytes.NewBufferString(
			"From: " + from + "\r\n" +
			"To: " + to + "\r\n" +
			"Subject: " + subject + "\r\n" +
			"Content-Type: " + contenttype + "\r\n" +
			"\r\n")
	if _, err := msg.Write(body); err != nil {
		return err
	}
	msg.WriteString("\r\n")
	log.Printf("Sending Mail From: %s, Subject: %s, To: %s", from, subject, to)
	err := smtp.SendMail(config.SMTPHost, auth, config.From, []string{to}, msg)
	return err
}

func manageMessage(msg *imap.Message, c *client.Client, config Config) error {
	from := msg.Envelope.Sender
	for _, addr := range from {
		for _, rule := range config.Rules {
			match, err := regexp.MatchString(rule.From, addr.Address())
			if err != nil {
				log.Printf("Regex Matching Error on Message %v", msg.SeqNum)
				log.Panic(err)
			}
			if match {
				log.Printf("Rule Matched on Message %v", msg.SeqNum)

				buf := bytes.NewBuffer(nil)
				contentType := "text/plain"

				for a, l := range msg.Body {
					if a.Specifier == "TEXT" {
						read, err := ioutil.ReadAll(l)
						if err != nil {
							log.Print(err)
						} else {
							buf.Write(read)
							buf.WriteString("\r\n")
						}
					} else {
						reader := bufio.NewReader(l)
						tp := textproto.NewReader(reader)

						mimeHeader, err := tp.ReadMIMEHeader()
						if err != nil {
							log.Fatal(err)
						}
						contentType = mimeHeader.Get("Content-Type")

					}
				}
				fromToUse := fmt.Sprint(config.FromName, " <",config.From,">")
				if !rule.RewriteFrom {
					fromToUse = addr.Address()
				}
				err := sendMail(config, fromToUse, msg.Envelope.Subject, rule.To, contentType, buf.Bytes())
				if err != nil {
					log.Print("Mail Sending Failed:")
					return err
				} else {
					log.Print("Email ", msg.SeqNum, " sent successfully, moving to trash")
					seqset := new(imap.SeqSet)
					seqset.AddNum(msg.SeqNum)
					c.Move(seqset, config.TrashMailbox)
				}
				return nil
			}
		}
	}
	return nil
}

func main() {
	config, err := readConfig(os.Args[1])
	if err != nil {
		log.Panic(err)
	}
	fmt.Printf("config: %v\n", config)
	c, err := mailConnection(config.IMAPHost, config.Login.Username, config.Login.Password)
	if err != nil {
		log.Panic(err)
	}
	defer c.Logout()
	// List mailboxes
	mailboxes := make(chan *imap.MailboxInfo, 10)
	done := make(chan error, 1)
	go func() {
		done <- c.List("", "*", mailboxes)
	}()

	log.Println("Mailboxes:")
	for m := range mailboxes {
		log.Println("* " + m.Name)
	}
	if err := <-done; err != nil {
		log.Panic(err)
	}

	mbox, err := c.Select(config.Mailbox, false)
	if err != nil {
		log.Panic(err)
	}

	from := uint32(1)
	to := mbox.Messages
	seqset := new(imap.SeqSet)
	seqset.AddRange(from, to)

	messages := make(chan *imap.Message, 10)
	done = make(chan error, 1)
	go func() {
		done <- c.Fetch(seqset, []imap.FetchItem{imap.FetchEnvelope, "BODY[TEXT]", "BODY[HEADER]"}, messages)
	}()
	for message := range messages {
		log.Println("Managing Message: ", message.SeqNum)
		err := manageMessage(message, c, config); if err != nil {
		log.Panic(err)
		}
	}

	if err := <-done; err != nil {
		log.Panic(err)
	}

	log.Println("Done!")

}
