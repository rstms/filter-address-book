package main

import (
	"errors"
	"github.com/rstms/rspamd-classes/classes"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/poolpOrg/OpenSMTPD-framework/filter"
)

const Version = "0.0.5"

/*********************************************************************************************

 filter-rspamd-class

 add a header: 'X-Spam-Class: SPAMCLASS'
 based on threshold levels compared with rspamd's X-Spam-Score header

 Default class threshold levels:

 ham:		spam_score < HAM_THRESHOLD
 possible:	HAM_THRESHOLD <= spam_score < POSSIBLE_THRESHOLD
 probable:	POSSIBLE_THRESHOLD <= spam_score < PROBABLE_THRESHOLD
 spam:		spam_score >= PROBABLE_THRESHOLD

 class names and thresholds are configurable per recipient email address
 using a JSON file with the following format:

{
    "username@example.org": [
	{ "name": "ham", "score": 0 },
	{ "name": "possible", "score": 3 },
	{ "name": "probable", "score": 10 },
	{ "name": "spam", "score": 999 }
    ],
    "othername@example.org": [
	{ "name": "not_spam", "score": 0 },
	{ "name": "suspected_spam", "score": 10 },
	{ "name": "is_spam", "score": 999 }
    ]
}

The final threshold value is set to float32-max automatically; 999 is a placeholder

*********************************************************************************************/

const Version = "0.1.17"

const CLASS_CONFIG_FILE = "/etc/mail/filter_rspamd_classes.json"

/*
type tx struct {
	msgid    string
	mailFrom string
	rcptTo   []string
	message  []string
	action   string
	response string
}

type session struct {
	id string

	rdns     string
	src      string
	heloName string
	userName string
	mtaName  string

	tx tx
}
*/

type SessionData struct {
	rcptTo []string
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
	sessionData.rcptTo = []string{}
	return nil
}

func txResetCb(timestamp time.Time, session filter.Session, messageId string) {
	err := clearSessionData(session)
	if err != nil {
		log.Printf("%s: %s: tx-reset error: %v\n", timestamp, session, err)
		return
	}
	//log.Printf("%s: %s: tx-reset: %s\n", timestamp, session, messageId)
}

func txBeginCb(timestamp time.Time, session filter.Session, messageId string) {
	err := clearSessionData(session)
	if err != nil {
		log.Printf("%s: %s: tx-begin error: %v\n", timestamp, session, err)
		return
	}
	//log.Printf("%s: %s: tx-begin: %s\n", timestamp, session, messageId)
}

func txRcptCb(timestamp time.Time, session filter.Session, messageId string, result string, to string) {
	sessionData, err := getSessionData(session)
	if err != nil {
		log.Printf("%s: %s: tx-rcpt error: %v\n", timestamp, session, err)
		return
	}
	sessionData.rcptTo = append(sessionData.rcptTo, to)
	//log.Printf( "%s: %s: tx-rcpt: %s|%s|%s\n", timestamp, session, messageId, result, to)
}

func parseSpamScore(line string) (float32, error) {
	fields := strings.Split(line, " ")
	if len(fields) < 2 {
		return 0, errors.New("spam score parse failed")
	}
	score, err := strconv.ParseFloat(fields[1], 32)
	if err != nil {
		return 0, err
	}
	return float32(score), nil
}

func filterDataLineCb(timestamp time.Time, session filter.Session, line string) []string {
	output := []string{line}
	if strings.HasPrefix(line, "X-Spam-Score: ") {
		sessionData, err := getSessionData(session)
		if err != nil {
			log.Printf("%s: %s: filter-data-line error: %v\n", timestamp, session, err)
			return output
		}
		score, err := parseSpamScore(line)
		if err != nil {
			log.Printf("%s: %s: filter-data-line error: %v\n", timestamp, session, err)
			return output
		}
		class := readClasses().GetClass(sessionData.rcptTo, score)
		if class != "" {
			output = append(output, "X-Spam-Class: "+class)
		}
		log.Printf("%s: %s: score=%v class='%s'\n", timestamp, session, score, class)
	}
	return output
}

func readClasses() *classes.SpamClasses {
	spamClasses, err := classes.New(CLASS_CONFIG_FILE)
	if err != nil {
		log.Fatalf("SpamClasses: config error: %v\n", err)
	}
	return spamClasses
}

func main() {
	log.SetFlags(0)
	log.Printf("Starting %s v%s rspamd_classes=v%s uid=%d gid=%d\n", os.Args[0], Version, classes.Version, os.Getuid(), os.Getgid())

	// read classes to report config error on startup
	readClasses()

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
