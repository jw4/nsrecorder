package nsrecorder // import "jw4.us/nsrecorder"

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"

	nsq "github.com/nsqio/go-nsq"
	"jw4.us/nspub"
)

func NewWatcher(ctx context.Context, store Store) *Watcher {
	w := &Watcher{ctx: ctx, store: store, msg: make(chan *nsq.Message)}
	w.start()
	return w
}

type Watcher struct {
	ctx      context.Context
	store    Store
	msg      chan *nsq.Message
	consumer *nsq.Consumer
}

func (w *Watcher) HandleMessage(message *nsq.Message) error {
	message.Touch()
	w.log(message)
	return nil
}

func (w *Watcher) Stop() {
	<-w.consumer.StopChan
}

func (w *Watcher) start() {
	topic, ok := w.ctx.Value("topic").(string)
	if !ok {
		panic("no topic")
	}

	channel, ok := w.ctx.Value("channel").(string)
	if !ok {
		panic("no channel")
	}

	lookupd, ok := w.ctx.Value("lookupd").([]string)
	if !ok {
		panic("no lookupd")
	}

	config := nsq.NewConfig()
	config.ClientID = "nsr"
	config.Hostname, _ = os.Hostname()
	config.UserAgent = "nsr go client"
	config.MaxInFlight = 10

	var err error
	if w.consumer, err = nsq.NewConsumer(topic, channel, config); err != nil {
		panic(err)
	}
	w.consumer.AddConcurrentHandlers(w, 10)
	go w.loop()
	if err = w.consumer.ConnectToNSQLookupds(lookupd); err != nil {
		panic(err)
	}
}

func (w *Watcher) loop() {
	messages := []*nsq.Message{}
	for {
		after := time.After(5 * time.Second)
		select {
		case msg := <-w.msg:
			messages = append(messages, msg)
		case <-after:
			if len(messages) > 0 {
				w.handleBatch(messages)
				messages = []*nsq.Message{}
			}
		case <-w.ctx.Done():
			w.consumer.Stop()
			return
		}
	}
}

func (w *Watcher) handleBatch(messages []*nsq.Message) {
	fmt.Printf("Processing: %v\n", time.Now())
	clients := []Client{}
	lookups := []Lookup{}
	for _, msg := range messages {
		c, l, err := parse(msg)
		if err != nil {
			msg.Requeue(-1)
			continue
		}
		clients = append(clients, c)
		lookups = append(lookups, l)
	}
	err := w.store.Accept(clients, lookups)
	for _, msg := range messages {
		switch err {
		case nil:
			msg.Finish()
		default:
			msg.Requeue(-1)
		}
	}
}

func (w *Watcher) log(message *nsq.Message) {
	w.msg <- message
}

func parse(rawmsg *nsq.Message) (Client, Lookup, error) {
	var (
		client Client
		lookup Lookup
		err    error

		hosts, qhosts, aips []string
	)

	msg := nspub.Message{}
	if err = json.Unmarshal(rawmsg.Body, &msg); err != nil {
		log.Printf("unmarshaling nsq message: %v\n%s", err, rawmsg.Body)
		return client, lookup, err
	}

	client.IP = msg.ClientIP
	client.Name = msg.ClientIP
	if hosts, err = net.LookupAddr(msg.ClientIP); err == nil {
		client.Name = hosts[0]
	}

	lookup.When = msg.Time
	lookup.Client = msg.ClientIP
	for _, q := range msg.Msg.Question {
		qhosts = append(qhosts, q.Name)
	}
	lookup.Host = strings.Join(qhosts, ",")

	for _, a := range msg.Msg.Answer {
		if a.A != "" {
			aips = append(aips, a.A)
		}
		if a.AAAA != "" {
			aips = append(aips, a.AAAA)
		}
	}
	if len(aips) > 0 {
		lookup.FirstIP = aips[0]
	}
	return client, lookup, nil
}
