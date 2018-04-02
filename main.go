package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	_ "github.com/lib/pq"
	nsq "github.com/nsqio/go-nsq"
	"jw4.us/nspub"
)

const (
	createTables = `
CREATE TABLE IF NOT EXISTS clients (ip STRING PRIMARY KEY, name STRING);
CREATE TABLE IF NOT EXISTS lookups (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), evt TIMESTAMP WITH TIME ZONE, clientip STRING, type STRING, name STRING, ips STRING[]);
`
	upsertClient = `
UPSERT INTO clients (ip, name) VALUES ($1, $2);
`
	upsertLookup = `
UPSERT INTO lookups (evt, clientip, type, name, ips) VALUES ($1, $2, $3, $4, $5);
`
	latestLookups = `
SELECT c.name, l.name, l.type, l.ips[1] as firstip, l.evt
FROM lookups AS l JOIN clients AS c ON l.clientip = c.ip
ORDER BY l.evt DESC
LIMIT $1
`
)

var (
	channel = "recorder"
	addr    = "localhost:4150"
	topic   = "dns"

	dbhost = "localhost"
	dbport = "26257"
	dbname = "recorder"
	dbuser = "recorder"
	dbpass = ""
)

func init() {
	flag.StringVar(&topic, "topic", topic, "NSQ topic name")
	flag.StringVar(&channel, "channel", channel, "NSQ channel name")
	flag.StringVar(&addr, "address", addr, "NSQ tcp address")

	flag.StringVar(&dbhost, "host", dbhost, "database host name")
	flag.StringVar(&dbport, "port", dbport, "database port number")
	flag.StringVar(&dbname, "name", dbname, "database name")
	flag.StringVar(&dbuser, "user", dbuser, "database user")
	flag.StringVar(&dbpass, "pass", dbpass, "database password")
}

func main() {
	var (
		db      *sql.DB
		err     error
		lookups []lookup
	)

	flag.Parse()

	if dbpass != "" {
		dbpass = ":" + dbpass
	}
	conn := fmt.Sprintf("postgresql://%s%s@%s:%s/%s?sslmode=disable", dbuser, dbpass, dbhost, dbport, dbname)

	if db, err = sql.Open("postgres", conn); err != nil {
		log.Fatalf("connecting to database: %v", err)
	}

	if err = initialize(db); err != nil {
		log.Fatalf("initializing db: %v", err)
	}

	if false {
		if lookups, err = getRecentLookups(db, 50); err != nil {
			log.Fatalf("getting recent events: %v", err)
		}

		for _, ll := range lookups {
			fmt.Printf("%s\n", ll.String())
		}
	}

	if err := processEvents(db); err != nil {
		log.Fatalf("processing events: %v", err)
	}
}

type lookup struct {
	When    time.Time
	Client  string
	Host    string
	Type    string
	FirstIP string
}

func (l *lookup) String() string {
	return fmt.Sprintf("%s\t%16s\t%-3s\t%-20s\t%s", l.When.Format(time.RFC3339), l.Client, l.Type, l.Host, l.FirstIP)
}

func initialize(db *sql.DB) error {
	var err error
	if _, err = db.Exec(createTables); err != nil {
		log.Printf("creating tables: %v", err)
		return err
	}
	return nil
}

func getRecentLookups(db *sql.DB, max int) ([]lookup, error) {
	var (
		rows *sql.Rows
		err  error
	)
	if max == 0 {
		max = 30
	}
	if rows, err = db.Query(latestLookups, max); err != nil {
		log.Printf("querying tables: %v", err)
		return nil, err
	}
	defer rows.Close()

	res := []lookup{}
	for rows.Next() {
		ll := lookup{}
		if err = rows.Scan(&ll.Client, &ll.Host, &ll.Type, &ll.FirstIP, &ll.When); err != nil {
			log.Printf("reading results: %v", err)
			return nil, err
		}
		res = append(res, ll)
	}
	return res, nil
}

func insertMessage(db *sql.DB, msg *nspub.Message) error {
	var (
		err error

		hosts, qhosts, aips    []string
		client, ip, typ, qhost string
	)

	ip = msg.ClientIP
	client = ip
	if hosts, err = net.LookupAddr(ip); err == nil {
		client = hosts[0]
	}
	if _, err = db.Exec(upsertClient, ip, client); err != nil {
		log.Printf("upserting client: %v", err)
		return err
	}

	for _, q := range msg.Msg.Question {
		qhosts = append(qhosts, q.Name)
	}
	qhost = strings.Join(qhosts, ",")
	for _, a := range msg.Msg.Answer {
		if a.A != "" {
			aips = append(aips, a.A)
		}
		if a.AAAA != "" {
			aips = append(aips, a.AAAA)
		}
	}
	if _, err = db.Exec(upsertLookup, msg.Time, msg.ClientIP, typ, qhost, fmt.Sprintf(`{"%s"}`, strings.Join(aips, `","`))); err != nil {
		log.Printf("upserting lookup: %v", err)
		return err
	}
	return nil
}

func processEvents(db *sql.DB) error {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	config := nsq.NewConfig()
	config.UserAgent = "recorder"
	consumer, err := nsq.NewConsumer(topic, channel, config)
	if err != nil {
		log.Printf("creating nsq consumer: %v", err)
		return err
	}
	consumer.AddHandler(&client{db: db})
	err = consumer.ConnectToNSQD(addr)
	if err != nil {
		log.Printf("connecting to nsq: %v", err)
		return err
	}

	<-sigChan
	consumer.Stop()
	<-consumer.StopChan

	return nil
}

type client struct {
	db *sql.DB
}

func (c *client) HandleMessage(rawmsg *nsq.Message) error {
	rawmsg.Touch()
	msg := nspub.Message{}
	if err := json.Unmarshal(rawmsg.Body, &msg); err != nil {
		log.Printf("unmarshaling nsq message: %v\n%s", err, rawmsg.Body)
		rawmsg.Requeue(5 * time.Second)
		return err
	}
	if err := insertMessage(c.db, &msg); err != nil {
		log.Printf("inserting nsq message: %v\n%s", err, rawmsg.Body)
		rawmsg.Requeue(5 * time.Second)
		return err
	}
	rawmsg.Finish()
	return nil
}
