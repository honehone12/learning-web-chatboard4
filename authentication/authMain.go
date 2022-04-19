package main

import (
	"learning-web-chatboard4/common"
	"learning-web-chatboard4/common/models"
	"learning-web-chatboard4/jose"
	"learning-web-chatboard4/rabbitrpc"
	"log"

	"xorm.io/xorm"
)

const audienceName = "chatboard4"

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

	//jose
	err = jose.StartJoseMaker("chatboard4-authentication-server")
	if err != nil {
		common.LogError(logger).Fatalln(err.Error())
	}
	jose.AddKnownAudience(audienceName)

	//rabbit
	server = rabbitrpc.NewRPCServer(
		rabbitrpc.DefaultRabbitURL,
		config.UsersResQName,
		config.UsersReqQName,
		config.UsersExchangeName,
		rabbitrpc.ExchangeKindDirect,
		config.UsersClientKey,
		config.UsersServerKey,
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
	// check data type
	switch envelop.DataTypeName {

	case "User":
		var user models.User
		err = envelop.Extract(&user)
		if err != nil {
			return
		}

		// check function name
		switch envelop.FunctionToCall {
		case "createUser":
			createUser(&user, corrId)
		case "readUser":
			readUser(&user, corrId)
		case "lockUser":
			lockUser(&user, corrId)
		default:
			err = rabbitrpc.ErrorFunctionNotFound
		}

	case "Token":
		var token common.Token
		err = envelop.Extract(&token)
		if err != nil {
			return
		}

		switch envelop.FunctionToCall {
		case "verifyToken":
			verifyToken(&token, corrId)
		default:
			err = rabbitrpc.ErrorFunctionNotFound
		}

	default:
		err = rabbitrpc.ErrorTypeNotFound
	}
	return
}
