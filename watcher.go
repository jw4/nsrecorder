package nsrecorder // import "jw4.us/nsrecorder"

import (
	"context"
	"fmt"
	"os"
	"time"

	nsq "github.com/nsqio/go-nsq"
)

func NewWatcher(ctx context.Context) *Watcher {
	w := &Watcher{ctx: ctx, msg: make(chan *nsq.Message)}
	w.start()
	return w
}

type Watcher struct {
	ctx      context.Context
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
	for ix, msg := range messages {
		fmt.Printf("%d: %x\n", ix, msg)
		msg.Finish()
	}
}

func (w *Watcher) log(message *nsq.Message) {
	w.msg <- message
}
