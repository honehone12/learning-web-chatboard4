package main

import (
	"learning-web-chatboard4/common"
	"learning-web-chatboard4/rabbitrpc"
	"learning-web-chatboard4/session"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

var config *common.Configuration
var logger *log.Logger
var usersClient *rabbitrpc.RabbitClient
var topicsClient *rabbitrpc.RabbitClient
var callbackPool rabbitrpc.CallbackPool
var callbackCh chan rabbitrpc.Raws
var doneCh chan string
var validate *validator.Validate

func main() {
	var err error

	// config
	config, err = common.LoadConfig()
	if err != nil {
		log.Fatalln(err.Error())
	}

	//session
	err = session.StartSessionMaker(
		config.UseSecureCookie,
		config.SetHttpOnlyCookie,
		config.ShowSQL,
	)
	if err != nil {
		log.Fatalln(err.Error())
	}

	//log
	logger, err = common.OpenLogger(
		config.LogToFile,
		config.LogFileNameThreads,
	)
	if err != nil {
		log.Fatal(err.Error())
	}

	//rabbit
	callbackCh = make(chan rabbitrpc.Raws)
	callbackPool = make(rabbitrpc.CallbackPool)
	doneCh = make(chan string)

	usersClient = rabbitrpc.NewRPCClient(
		rabbitrpc.DefaultRabbitURL,
		config.UsersReqQName,
		config.UsersResQName,
		config.UsersExchangeName,
		rabbitrpc.ExchangeKindDirect,
		config.UsersServerKey,
		config.UsersClientKey,
		func(raws rabbitrpc.Raws) {
			callbackCh <- raws
		},
	)
	defer usersClient.Publisher.Done()
	defer usersClient.Subscriber.Done()

	topicsClient = rabbitrpc.NewRPCClient(
		rabbitrpc.DefaultRabbitURL,
		config.TopicsReqQName,
		config.TopicsResQName,
		config.TopicsExchangeName,
		rabbitrpc.ExchangeKindDirect,
		config.TopicsServerKey,
		config.TopicsClientKey,
		func(raws rabbitrpc.Raws) {
			callbackCh <- raws
		},
	)
	defer topicsClient.Publisher.Done()
	defer topicsClient.Subscriber.Done()

	go func() {
	loop:
		for {
			select {
			case <-usersClient.Publisher.CTX.Done():
				break loop
			case <-usersClient.Subscriber.CTX.Done():
				break loop
			case <-topicsClient.Publisher.CTX.Done():
				break loop
			case <-topicsClient.Subscriber.CTX.Done():
				break loop
			case raws := <-callbackCh:
				fn, ok := callbackPool[raws.CorrelationId]
				if ok {
					go fn(raws)
				} else {
					common.LogError(logger).Println("recieved unknown response")
				}
			case id := <-doneCh:
				delete(callbackPool, id)
			}
		}
	}()

	// validator
	validate = validator.New()

	//gin
	webEngine := gin.Default()
	// setup templates
	webEngine.Static("/static", "./public")
	webEngine.Delims("{{", "}}")
	webEngine.LoadHTMLGlob("./templates/*")
	//setup routes
	webEngine.GET(
		"/",
		setCommonHeadersMiddleware,
		sessionCheckMiddleware,
		loggedInCheckMiddleware,
		indexGet,
	)
	webEngine.GET(
		"/error",
		setCommonHeadersMiddleware,
		sessionCheckMiddleware,
		loggedInCheckMiddleware,
		errorGet,
	)

	usersRoute := webEngine.Group("/user")
	usersRoute.Use(
		setCommonHeadersMiddleware,
		sessionCheckMiddleware,
		loggedInCheckMiddleware,
	)
	usersRoute.GET(
		"/login",
		generateSessionStateMiddleware,
		loginGet,
	)
	usersRoute.GET(
		"/signup",
		generateSessionStateMiddleware,
		signupGet,
	)
	usersRoute.POST("/logout", logoutPost)
	usersRoute.POST("/signup-account", signupPost)
	usersRoute.POST("/authenticate", authenticatePost)

	threadsRoute := webEngine.Group("/topic")
	threadsRoute.Use(
		setCommonHeadersMiddleware,
		sessionCheckMiddleware,
		loggedInCheckMiddleware,
	)
	threadsRoute.GET(
		"/read",
		generateSessionStateMiddleware,
		topicGet,
	)
	threadsRoute.GET(
		"/new",
		generateSessionStateMiddleware,
		newTopicGet,
	)
	threadsRoute.POST("/create", newTopicPost)
	threadsRoute.POST("/post", newReplyPost)

	webEngine.Run(config.AddressRouter)
}
