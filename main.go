package main

import (
	"errors"
	"fmt"
	"github.com/poolpOrg/OpenSMTPD-framework/filter"
	flag "github.com/spf13/pflag"
	"log"
	"os"
	"strings"
	"time"
)

const Version = "0.1.15"

var Verbose bool

const DEFAULT_CONFIG_FILE = "/etc/mail/filter-address-book.yml"

/*********************************************************************************************

 filter-address-book

 Check sender address of email and scan for it in the recipient's CardDAV address books.
 If the sender address is present in an address book, add an X-Address-Book header with
 the value set to the name of the address book.

*********************************************************************************************/

type SessionData struct {
	From   string
	To     []string
	Client *FilterControlClient
}

func getSessionData(timestamp time.Time, label string, session filter.Session) (*SessionData, error) {
	data := session.Get()
	sessionData, ok := data.(*SessionData)
	if !ok {
		return nil, errors.New("SessionData conversion failure")
	}
	if Verbose {
		log.Printf("%s: %s %s session=%+v\n", timestamp, label, session)
	}
	return sessionData, nil
}

func clearSessionData(timestamp time.Time, label string, session filter.Session) error {
	sessionData, err := getSessionData(timestamp, label+"-clear-session-data", session)
	if err != nil {
		return err
	}
	sessionData.From = ""
	sessionData.To = []string{}
	sessionData.Client = NewFilterControlClient()
	return nil
}

func txResetCb(timestamp time.Time, session filter.Session, messageId string) {
	err := clearSessionData(timestamp, "tx-reset", session)
	if err != nil {
		log.Printf("%s: %s: tx-reset error: %v\n", timestamp, session, err)
		return
	}
	log.Printf("%s: %s: tx-reset: %s\n", timestamp, session, messageId)
}

func txBeginCb(timestamp time.Time, session filter.Session, messageId string) {
	err := clearSessionData(timestamp, "tx-begin", session)
	if err != nil {
		log.Printf("%s: %s: tx-begin error: %v\n", timestamp, session, err)
		return
	}
	log.Printf("%s: %s: tx-begin: %s\n", timestamp, session, messageId)
}

func txRcptCb(timestamp time.Time, session filter.Session, messageId string, result string, to string) {
	sessionData, err := getSessionData(timestamp, "tx-rcpt", session)
	if err != nil {
		log.Printf("%s: %s: tx-rcpt error: %v\n", timestamp, session, err)
		return
	}
	sessionData.To = append(sessionData.To, to)
	log.Printf("%s: %s: tx-rcpt: %s|%s|%s\n", timestamp, session, messageId, result, to)
}

func filterDataLineCb(timestamp time.Time, session filter.Session, line string) []string {
	output := []string{line}

	if strings.HasPrefix(line, "X-Address-Book:") {
		// remove existing X-Address-Book header
		return output

	}
	if Verbose {
		log.Printf("%s: %s: filter-data-line line: %s\n", timestamp, session, line)
	}
	if strings.HasPrefix(line, "From:") {
		sessionData, err := getSessionData(timestamp, "filter-data-line", session)
		if err != nil {
			log.Printf("%s: %s: filter-data-line error: %v\n", timestamp, session, err)
			return output
		}
		fromAddress, err := parseEmailAddress(line)
		if err != nil {
			log.Printf("%s: %s: filter-data-line error: %v\n", timestamp, session, err)
			return output
		}
		for _, recipient := range sessionData.To {
			toAddress, err := parseEmailAddress(recipient)
			if err != nil {
				log.Printf("%s: %s: filter-data-line error: %v\n", timestamp, session, err)
				continue
			}
			log.Printf("%s: %s: filter-data-line lookup sender=%s recipient=%s\n", timestamp, session, fromAddress, toAddress)
			books, err := sessionData.Client.ScanAddressBooks(toAddress, fromAddress)
			if err != nil {
				log.Printf("%s: %s: filter-data-line error: %v\n", timestamp, session, err)
				continue
			}
			if len(books) > 0 {
				value := strings.Join(books, ",")
				header := "X-Address-Book: " + value
				output = append(output, header)
				log.Printf("%s: %s: add-header: '%s'\n", timestamp, session, header)
			}
		}
	}
	return output
}

func main() {

	versionFlag := flag.Bool("version", false, "output version")
	verboseFlag := flag.Bool("verbose", false, "enable diagnostic log output")
	helpFlag := flag.Bool("help", false, "show help")

	flag.Parse()

	if *helpFlag {
		flag.Usage()
		os.Exit(0)
	}

	if *versionFlag {
		fmt.Printf("filter-address-book version %s\n", Version)
		os.Exit(0)
	}

	if *verboseFlag {
		Verbose = true
	}

	log.SetFlags(0)
	log.Printf("Starting %s v%s uid=%d gid=%d\n", os.Args[0], Version, os.Getuid(), os.Getgid())

	err := Configure(DEFAULT_CONFIG_FILE)
	if err != nil {
		log.Fatalf("configuration failed: %v\n", err)
	}

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
