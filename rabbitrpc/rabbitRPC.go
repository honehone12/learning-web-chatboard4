package rabbitrpc

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	DefaultRabbitURL = "amqp://guest:guest@localhost:5672"

	ExchangeKindFanout = "fanout"
	ExchangeKindDirect = "direct"
	ExchangeKindTopic  = "topic"
)

var rabbitLogger *log.Logger

type Raws struct {
	Body          []byte
	CorrelationId string
}

type session struct {
	*amqp.Connection
	*amqp.Channel
}

func (sess *session) close() error {
	if sess.Connection == nil {
		return nil
	}
	return sess.Connection.Close()
}

type RabbitHandle struct {
	CTX  context.Context
	Done context.CancelFunc
	Ch   chan Raws
}

type RabbitClient struct {
	Publisher  *RabbitHandle
	Subscriber *RabbitHandle

	ContentType         string
	RabbitURL           string
	PublishQueueName    string
	SubscribeQueueName  string
	ExchangeName        string
	ExchangeKind        string
	PublishRoutingKey   string
	SubscribeRoutingKey string
}

func openLogger() {
	if rabbitLogger == nil {
		rabbitLogger = log.New(
			os.Stdout,
			"[RabbitMQ RPC] ",
			log.Ldate|log.Ltime|log.Lshortfile,
		)
	}
}

func NewRPCClient(
	rabbitURL string,
	publishQueueName string,
	subscribeQueueName string,
	exchangeName string,
	exchangeKind string,
	publishKey string,
	subscribeKey string,
	callback func(raws Raws),
) (client *RabbitClient) {
	client = &RabbitClient{
		ContentType:         "application/json",
		RabbitURL:           rabbitURL,
		PublishQueueName:    publishQueueName,
		SubscribeQueueName:  subscribeQueueName,
		ExchangeName:        exchangeName,
		ExchangeKind:        exchangeKind,
		PublishRoutingKey:   publishKey,
		SubscribeRoutingKey: subscribeKey,
	}

	client.Publisher = &RabbitHandle{}
	client.Publisher.CTX, client.Publisher.Done = context.WithCancel(
		context.Background(),
	)
	client.Publisher.Ch = make(chan Raws)

	client.Subscriber = &RabbitHandle{}
	client.Subscriber.CTX, client.Subscriber.Done = context.WithCancel(
		context.Background(),
	)

	go func() {
		client.publisherRoutine(
			redial(
				client.Publisher.CTX,
				client.RabbitURL,
				client.ExchangeName,
				client.ExchangeKind,
			),
			client.Publisher.Ch,
		)
	}()

	go func() {
		client.subscriberRoutine(
			redial(
				client.Subscriber.CTX,
				client.RabbitURL,
				client.ExchangeName,
				client.ExchangeKind,
			),
			setCallback(callback),
		)
	}()

	openLogger()
	return
}

func NewRPCServer(
	rabbitURL string,
	publishQueueName string,
	subscribeQueueName string,
	exchangeName string,
	exchangeKind string,
	publishKey string,
	subscribeKey string,
	callback func(raws Raws),
) (server *RabbitClient) {
	server = &RabbitClient{
		ContentType:         "application/json",
		RabbitURL:           rabbitURL,
		PublishQueueName:    publishQueueName,
		SubscribeQueueName:  subscribeQueueName,
		ExchangeName:        exchangeName,
		ExchangeKind:        exchangeKind,
		PublishRoutingKey:   publishKey,
		SubscribeRoutingKey: subscribeKey,
	}

	server.Publisher = &RabbitHandle{}
	server.Publisher.CTX, server.Publisher.Done = context.WithCancel(
		context.Background(),
	)
	server.Publisher.Ch = make(chan Raws)

	server.Subscriber = &RabbitHandle{}
	server.Subscriber.CTX, server.Subscriber.Done = context.WithCancel(
		context.Background(),
	)

	go func() {
		server.publisherRoutine(
			redial(
				server.Publisher.CTX,
				server.RabbitURL,
				server.ExchangeName,
				server.ExchangeKind,
			),
			server.Publisher.Ch,
		)
	}()

	go func() {
		server.subscriberRoutine(
			redial(
				server.Subscriber.CTX,
				server.RabbitURL,
				server.ExchangeName,
				server.ExchangeKind,
			),
			setCallback(callback),
		)
	}()

	openLogger()
	return
}

func (rabbit *RabbitClient) GenerateCorrelationID() string {
	return fmt.Sprintf(
		"%s/at%d/%s.to.%s",
		rabbit.ExchangeName,
		time.Now().UnixMicro(),
		rabbit.SubscribeRoutingKey,
		rabbit.PublishRoutingKey,
	)
}

func redial(
	ctx context.Context,
	url string,
	exchangeName string,
	exchangeKind string,
) chan chan session {
	sessions := make(chan chan session)

	go func() {
		sess := make(chan session)

		for {
			select {
			case sessions <- sess:
			case <-ctx.Done():
				rabbitLogger.Println("shutting down session factory")
				return
			}

			conn, err := amqp.Dial(url)
			if err != nil {
				rabbitLogger.Fatalf("cannot (re)dial: %v %q", err, url)
			}

			ch, err := conn.Channel()
			if err != nil {
				rabbitLogger.Fatalf("cannot create channel: %v", err)
			}

			err = ch.Qos(
				1,
				0,
				false,
			)
			if err != nil {
				rabbitLogger.Fatalln("failed to set QoS")
			}

			err = ch.ExchangeDeclare(
				exchangeName,
				exchangeKind,
				false,
				true,
				false,
				false,
				nil,
			)
			if err != nil {
				rabbitLogger.Fatalf("cannot declare exchange: %v", err)
			}

			select {
			case sess <- session{conn, ch}:
			case <-ctx.Done():
				rabbitLogger.Println("shutting down new session")
				return
			}
		}
	}()
	return sessions
}

// publisher

func (rabbit *RabbitClient) publisherRoutine(
	sessions chan chan session, messages <-chan Raws) {

	for sess := range sessions {
		var (
			isRunning bool
			readingCh = messages
			pendingCh = make(chan Raws, 1)
			confirmCh = make(chan amqp.Confirmation, 1)
		)

		pub := <-sess
		err := pub.Confirm(false)
		if err != nil {
			rabbitLogger.Printf("publisher confirms not supported %v", err)
			close(confirmCh)
		} else {
			pub.NotifyPublish(confirmCh)
		}
		rabbitLogger.Printf("publishing...")

	publishLoop:
		for {
			var raws Raws

			select {
			case confirmed, ok := <-confirmCh:
				if !ok {
					break publishLoop
				}
				if !confirmed.Ack {
					rabbitLogger.Printf(
						"nack message %d, body: %q",
						confirmed.DeliveryTag,
						string(raws.Body),
					)
				}
				readingCh = messages
			case raws = <-pendingCh:
				err := pub.Publish(
					rabbit.ExchangeName,
					rabbit.PublishRoutingKey,
					false,
					false,
					amqp.Publishing{
						ContentType:   rabbit.ContentType,
						CorrelationId: raws.CorrelationId,
						Body:          raws.Body,
					},
				)
				if err != nil {
					pendingCh <- raws
					pub.close()
					break publishLoop
				}
			case raws, isRunning = <-readingCh:
				if !isRunning {
					return
				}
				pendingCh <- raws
				readingCh = nil
			}
		}
	}
}

// subscriber

func (rabbit *RabbitClient) subscriberRoutine(
	sessions chan chan session, messages chan<- Raws) {

	for sess := range sessions {
		sub := <-sess

		_, err := sub.QueueDeclare(
			rabbit.SubscribeQueueName,
			false,
			true,
			true,
			false,
			nil,
		)
		if err != nil {
			rabbitLogger.Printf(
				"cannot consume from exclusive queue: %q %v",
				rabbit.SubscribeQueueName,
				err,
			)
			return
		}

		err = sub.QueueBind(
			rabbit.SubscribeQueueName,
			rabbit.SubscribeRoutingKey,
			rabbit.ExchangeName,
			false,
			nil,
		)
		if err != nil {
			rabbitLogger.Printf(
				"cannot cosume without a binding to exchange: %q %v",
				rabbit.ExchangeName,
				err,
			)
			return
		}

		deliveries, err := sub.Consume(
			rabbit.SubscribeQueueName,
			"",
			false,
			true,
			false,
			false,
			nil,
		)
		if err != nil {
			rabbitLogger.Printf(
				"cannot consume from: %q %v",
				rabbit.SubscribeQueueName,
				err,
			)
			return
		}

		rabbitLogger.Printf("subscribed...")

		for deli := range deliveries {
			messages <- Raws{
				Body:          deli.Body,
				CorrelationId: deli.CorrelationId,
			}
			sub.Ack(deli.DeliveryTag, false)
		}
	}
}

func setCallback(callback func(raws Raws)) chan<- Raws {
	messages := make(chan Raws)
	go func() {
		for msg := range messages {
			callback(msg)
		}
	}()
	return messages
}
