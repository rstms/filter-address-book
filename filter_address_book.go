package main

import (
	"errors"
	"github.com/poolpOrg/OpenSMTPD-framework/filter"
	"github.com/rstms/mabctl/api"
	"github.com/spf13/viper"
	"log"
	"os"
	"strings"
	"time"
)

const Version = "0.0.5"

/*********************************************************************************************

 filter-address-book

 Check sender address of email and scan for it in the recipient's CardDAV address books.
 If the sender address is present in an address book, add an X-Address-Book header with
 the value set to the name of the address book.

*********************************************************************************************/

type SessionData struct {
	From string
	To   []string
}

func getSessionData(session filter.Session) (*SessionData, error) {
	data := session.Get()
	sessionData, ok := data.(*SessionData)
	if !ok {
		return nil, errors.New("SessionData conversion failure")
	}
	return sessionData, nil
}

func clearSessionData(session filter.Session) error {
	sessionData, err := getSessionData(session)
	if err != nil {
		return err
	}
	sessionData.From = ""
	sessionData.To = []string{}
	return nil
}

func txResetCb(timestamp time.Time, session filter.Session, messageId string) {
	err := clearSessionData(session)
	if err != nil {
		log.Printf("%s: %s: tx-reset error: %v\n", timestamp, session, err)
		return
	}
	log.Printf("%s: %s: tx-reset: %s\n", timestamp, session, messageId)
}

func txBeginCb(timestamp time.Time, session filter.Session, messageId string) {
	err := clearSessionData(session)
	if err != nil {
		log.Printf("%s: %s: tx-begin error: %v\n", timestamp, session, err)
		return
	}
	log.Printf("%s: %s: tx-begin: %s\n", timestamp, session, messageId)
}

func txRcptCb(timestamp time.Time, session filter.Session, messageId string, result string, to string) {
	sessionData, err := getSessionData(session)
	if err != nil {
		log.Printf("%s: %s: tx-rcpt error: %v\n", timestamp, session, err)
		return
	}
	sessionData.To = append(sessionData.To, to)
	log.Printf("%s: %s: tx-rcpt: %s|%s|%s\n", timestamp, session, messageId, result, to)
}

func MAB() (*api.Controller, error) {
	return api.NewAddressBookController()
}

func ScanAddressBooks(api *api.Controller, username, address string) ([]string, error) {
	response, err := api.ScanAddress(username, address)
	if err != nil {
		return []string{}, err
	}
	books := make([]string, len(response.Books))
	for i, book := range response.Books {
		books[i] = book.BookName
	}
	return books, nil
}

func filterDataLineCb(timestamp time.Time, session filter.Session, line string) []string {
	output := []string{line}
	if strings.HasPrefix(line, "From: ") {
		sessionData, err := getSessionData(session)
		if err != nil {
			log.Printf("%s: %s: filter-data-line error: %v\n", timestamp, session, err)
			return output
		}
		api, err := MAB()
		if err != nil {
			log.Printf("%s: %s: filter-data-line error: %v\n", timestamp, session, err)
			return output
		}
		for _, recipient := range sessionData.To {
			books, err := ScanAddressBooks(api, recipient, line[7:])
			if err != nil {
				log.Printf("%s: %s: filter-data-line error: %v\n", timestamp, session, err)
				return output
			}
			for _, book := range books {
				header := "X-Address-Book: " + book
				output = append(output, header)
				log.Printf("%s: %s: header='%s'\n", timestamp, session, header)
			}
		}
	}
	return output
}

func Configure() error {
	viper.SetConfigType("yaml")
	viper.SetConfigFile("/etc/mabctl/config")
	return viper.ReadInConfig()
}

func main() {
	log.SetFlags(0)
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("failed getting current directory: %v", err)
	}
	log.Printf("Starting %s v%s mabctl=%s uid=%d gid=%d cwd=%s\n", os.Args[0], Version, api.Version, os.Getuid(), os.Getgid(), cwd)

	filter.Init()

	filter.SMTP_IN.SessionAllocator(func() filter.SessionData {
		return &SessionData{}
	})

	filter.SMTP_IN.OnTxReset(txResetCb)
	filter.SMTP_IN.OnTxBegin(txBeginCb)
	filter.SMTP_IN.OnTxRcpt(txRcptCb)
	filter.SMTP_IN.DataLineRequest(filterDataLineCb)

	filter.Dispatch()
}
