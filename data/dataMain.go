package main

import (
	"learning-web-chatboard4/common"
	"learning-web-chatboard4/common/models"
	"learning-web-chatboard4/rabbitrpc"
	"log"

	"xorm.io/xorm"
)

var dbEngine *xorm.Engine
var config *common.Configuration
var logger *log.Logger
var server *rabbitrpc.RabbitClient

func main() {
	var err error

	// config
	config, err = common.LoadConfig()
	if err != nil {
		log.Fatalln(err.Error())
	}

	//log
	logger, err = common.OpenLogger(
		config.LogToFile,
		config.LogFileNameUsers,
	)
	if err != nil {
		log.Fatal(err.Error())
	}

	//database
	dbEngine, err = common.OpenDb(
		config.DbName,
		config.ShowSQL,
		0,
	)
	if err != nil {
		common.LogError(logger).Fatalln(err.Error())
	}

	//rabbit
	server = rabbitrpc.NewRPCServer(
		rabbitrpc.DefaultRabbitURL,
		config.TopicsResQName,
		config.TopicsReqQName,
		config.TopicsExchangeName,
		rabbitrpc.ExchangeKindDirect,
		config.TopicsClientKey,
		config.TopicsServerKey,
		onRequestReceived,
	)
	defer server.Publisher.Done()
	defer server.Subscriber.Done()

	select {
	case <-server.Publisher.CTX.Done():
		break
	case <-server.Subscriber.CTX.Done():
		break
	}
}

func onRequestReceived(raws rabbitrpc.Raws) {
	go func() {
		var err *rabbitrpc.RabbitRPCError
		envelop, err := rabbitrpc.FromBin(raws.Body)
		if err != nil {
			common.SendError(server, err, raws.CorrelationId)
			return
		}

		err = routingRequest(envelop, raws.CorrelationId)
		if err != nil {
			common.SendError(server, err, raws.CorrelationId)
		}
	}()
}

func routingRequest(envelop *rabbitrpc.Envelope, corrId string,
) (err *rabbitrpc.RabbitRPCError) {
	switch envelop.DataTypeName {

	case "Topic":
		var topic models.Topic
		err = envelop.Extract(&topic)
		if err != nil {
			return
		}

		switch envelop.FunctionToCall {

		case "createTopic":
			createTopic(&topic, corrId)
		case "readATopic":
			readATopic(&topic, corrId)
		case "readRepliesInTopic":
			readRepliesInTopic(&topic, corrId)
		case "readTopics":
			readTopics(corrId)
		case "updat	eTopic":
			updateTopic(&topic, corrId)
		case "incrementTopic":
			incrementTopic(&topic, corrId)
		default:
			err = rabbitrpc.ErrorFunctionNotFound
		}

	case "Reply":
		var reply models.Reply
		err = envelop.Extract(&reply)
		if err != nil {
			return
		}

		switch envelop.FunctionToCall {
		case "createReply":
			createReply(&reply, corrId)
		default:
			err = rabbitrpc.ErrorFunctionNotFound
		}

	default:
		err = rabbitrpc.ErrorTypeNotFound
	}
	return
}
