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

const Version = "0.1.17"

var Verbose bool

const DEFAULT_CONFIG_FILE = "/etc/mail/filter-address-book.yml"

/*********************************************************************************************

 filter-address-book

 Check sender address of email and scan for it in the recipient's CardDAV address books.
 If the sender address is present in an address book, add an X-Address-Book header with
 the value set to the name of the address book.

*********************************************************************************************/

type SessionData struct {
	EnvelopeFrom   string
	EnvelopeTo     []string
	HeaderFrom     string
	HeaderTo       string
	Client         *FilterControlClient
	HeaderComplete bool
}

func logError(session filter.Session, label string, err error) {
	logMessage(session, label, "error: %v", err)
}

func logMessage(session filter.Session, label, format string, args ...interface{}) {
	log.Printf("%s %s: %s\n", session, label, fmt.Sprintf(format, args...))
}

func loggedParseEmailAddress(session filter.Session, label, data string) (string, error) {
	address, err := parseEmailAddress(data)
	if err != nil {
		return "", fmt.Errorf("parseEmailAddress(%s) failed: %v", data, err)
	}
	logMessage(session, label, "parseEmailAddress(%s) returned: '%s'", data, address)
	return address, nil
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
	sessionData.EnvelopeFrom = ""
	sessionData.EnvelopeTo = []string{}
	sessionData.HeaderFrom = ""
	sessionData.HeaderTo = ""
	sessionData.Client = NewFilterControlClient()
	sessionData.HeaderComplete = false
	return nil
}

func txResetCb(timestamp time.Time, session filter.Session, messageId string) {
	label := "tx-reset"
	logMessage(session, label, "%s", messageId)
	err := clearSessionData(session)
	if err != nil {
		logError(session, label, err)
		return
	}
}

func txBeginCb(timestamp time.Time, session filter.Session, messageId string) {
	label := "tx-begin"
	logMessage(session, label, "%s", messageId)
	err := clearSessionData(session)
	if err != nil {
		logError(session, label, err)
		return
	}
}

func txMailCb(timestamp time.Time, session filter.Session, messageId string, result string, fromAddress string) {
	label := "tx-mail-from"
	logMessage(session, label, "%s|%s|%s", messageId, result, fromAddress)
	sessionData, err := getSessionData(session)
	if err != nil {
		logError(session, label, err)
		return
	}
	if sessionData.EnvelopeFrom != "" {
		logError(session, label, fmt.Errorf("redundant MAIL-FROM: %s", fromAddress))
	}
	sessionData.EnvelopeFrom = fromAddress
	logMessage(session, label, "EnvelopeFrom=%s", fromAddress)
}

func txRcptCb(timestamp time.Time, session filter.Session, messageId string, result string, toAddress string) {
	label := "tx-rcpt-to"
	logMessage(session, label, "%s|%s|%s", messageId, result, toAddress)
	sessionData, err := getSessionData(session)
	if err != nil {
		logError(session, label, err)
		return
	}
	sessionData.EnvelopeTo = append(sessionData.EnvelopeTo, toAddress)
	logMessage(session, label, "EnvelopeTo=%v", sessionData.EnvelopeTo)
}

func txCommitCb(timestamp time.Time, session filter.Session, messageId string, messageSize int) {
	label := "tx-commit"
	logMessage(session, label, "%s|%d", messageId, messageSize)
}

func filterDataLineCb(timestamp time.Time, session filter.Session, line string) []string {

	label := "filter-data-line"
	if Verbose {
		logMessage(session, label, "%s", line)
	}

	output := []string{line}

	if strings.HasPrefix(line, "X-Address-Book:") {
		// remove existing X-Address-Book header
		return []string{}

	}
	sessionData, err := getSessionData(session)
	if err != nil {
		logError(session, label, err)
		return output
	}
	switch {
	case strings.HasPrefix(line, "From:"):
		if sessionData.HeaderFrom != "" {
			logError(session, label, fmt.Errorf("redundant From header: %s", line))
			return output
		}
		fromAddress, err := loggedParseEmailAddress(session, label, line)
		if err != nil {
			logError(session, label, err)
			return output
		}
		sessionData.HeaderFrom = fromAddress
		logMessage(session, label, "HeaderFrom=%s", fromAddress)
	case strings.HasPrefix(line, "To:"):
		if sessionData.HeaderTo != "" {
			logError(session, label, fmt.Errorf("redundant To header: %s", line))
			return output
		}
		toAddress, err := loggedParseEmailAddress(session, label, line)
		if err != nil {
			logError(session, label, err)
			return output
		}
		sessionData.HeaderTo = toAddress
		logMessage(session, label, "HeaderTo=%s", toAddress)

	case strings.TrimSpace(line) == "" && sessionData.HeaderComplete == false:
		sessionData.HeaderComplete = true
		logMessage(session, label, "end-of-header sessionData=%+v", sessionData)
		/*
			for _, recipient := range sessionData.To {
				toAddress, err := loggedParseEmailAddress(session, label, recipient)
				if err != nil {
					logError(session, label, err)
					continue
				}
				log.Printf("%s: %s: filter-data-line lookup sender=%s recipient=%s\n", timestamp, session, fromAddress, toAddress)
				books, err := sessionData.Client.ScanAddressBooks(toAddress, fromAddress)
				if err != nil {
					logError(session, label, err)
					continue
				}
				if len(books) > 0 {
					value := strings.Join(books, ",")
					header := "X-Address-Book: " + value
					output = append(output, header)
					log.Printf("%s: %s: add-header: '%s'\n", timestamp, session, header)
				}
			}
		*/
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
	filter.SMTP_IN.OnTxMail(txMailCb)
	filter.SMTP_IN.OnTxCommit(txCommitCb)
	filter.SMTP_IN.DataLineRequest(filterDataLineCb)

	filter.Dispatch()
}
