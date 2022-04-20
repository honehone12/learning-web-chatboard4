package common

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"os"
	"runtime"
	"unicode/utf8"

	"learning-web-chatboard4/rabbitrpc"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
	"xorm.io/xorm"
)

type Configuration struct {
	AddressRouter string `json:"address_router"`

	UsersExchangeName string `json:"Users_exchange_name"`
	UsersReqQName     string `json:"users_req_q_name"`
	UsersServerKey    string `json:"users_server_key"`
	UsersResQName     string `json:"users_res_q_name"`
	UsersClientKey    string `json:"users_client_key"`

	SessionsExchangeName string `json:"sessions_exchange_name"`
	SessionsReqQName     string `json:"sessions_req_q_name"`
	SessionsServerKey    string `json:"sessions_server_key"`
	SessionsResQName     string `json:"sessions_res_q_name"`
	SessionsClientKey    string `json:"sessions_client_key"`

	TopicsExchangeName string `json:"topics_exchange_name"`
	TopicsReqQName     string `json:"topics_req_q_name"`
	TopicsServerKey    string `json:"topics_server_key"`
	TopicsResQName     string `json:"topics_res_q_name"`
	TopicsClientKey    string `json:"topics_client_key"`

	UseSecureCookie    bool   `json:"use_secure_cookie"`
	SetHttpOnlyCookie  bool   `json:"set_http_only_cookie"`
	DbName             string `json:"db_name"`
	ShowSQL            bool   `json:"show_sql"`
	LogToFile          bool   `json:"log_to_file"`
	LogFileNameRouter  string `json:"log_file_name_router"`
	LogFileNameUsers   string `json:"log_file_name_users"`
	LogFileNameThreads string `json:"log_file_name_threads"`
}

type SimpleMessage struct {
	Message string `json:"message"`
}

type Token struct {
	UserEmail string `json:"user_email"`
	Raw       string `json:"raw"`
}

const runeSource = "aA1bB2cC3dD4eE5fFgGhHiIjJkKlLm0MnNoOpPqQrRsStTuUvV6wW7xX8yY9zZ"

const (
	ConfigFileName = "../config.json"
	DbDriver       = "postgres"
	DbParameter    = "dbname=%s user=%s password=%s host=localhost port=5432 sslmode=disable"
)

const (
	LogInfoPrefix    = "[INFO]"
	LogWarningPrefix = "[WARNING]"
	LogErrorPrefix   = "[ERROR]"
)

// initialization

func LoadConfig() (config *Configuration, err error) {
	file, err := os.Open(ConfigFileName)
	if err != nil {
		return
	}
	decoder := json.NewDecoder(file)
	config = &Configuration{}
	err = decoder.Decode(config)
	if err != nil {
		return
	}
	return
}

// set maxConn<=0 if use default
func OpenDb(
	dbName string,
	showSQL bool,
	maxConn int,
) (dbEngine *xorm.Engine, err error) {
	dbEngine, err = xorm.NewEngine(
		DbDriver,
		fmt.Sprintf(
			DbParameter,
			dbName,
			os.Getenv("DBUSER"),
			os.Getenv("DBPASS"),
		),
	)
	if err != nil {
		return
	}
	dbEngine.ShowSQL(showSQL)
	if maxConn <= 0 {
		maxConn = runtime.NumCPU()
	}
	dbEngine.SetMaxOpenConns(maxConn)
	return
}

// Logging

func OpenLogger(logToFile bool, logFileName string) (logger *log.Logger, err error) {
	if logToFile {
		var file *os.File
		file, err = os.OpenFile(
			fmt.Sprintf("%s.log", logFileName),
			os.O_CREATE|os.O_WRONLY|os.O_APPEND,
			0666,
		)
		if err != nil {
			return
		}
		logger = log.New(
			file,
			LogInfoPrefix,
			log.Ldate|log.Ltime|log.Lshortfile,
		)
	} else {
		logger = log.New(
			os.Stdout,
			LogInfoPrefix,
			log.Ldate|log.Ltime|log.Lshortfile,
		)
	}
	return
}

func LogInfo(logger *log.Logger) *log.Logger {
	logger.SetPrefix(LogInfoPrefix)
	return logger
}

func LogWarning(logger *log.Logger) *log.Logger {
	logger.SetPrefix(LogWarningPrefix)
	return logger
}

func LogError(logger *log.Logger) *log.Logger {
	logger.SetPrefix(LogErrorPrefix)
	return logger
}

// helpers

// func ProcessPassword(pw, salt string, numStretching int) (hashed string) {
// 	// see these pkgs
// 	// https://pkg.go.dev/golang.org/x/crypto/bcrypt
// 	// https://pkg.go.dev/golang.org/x/crypto/scrypt

// 	asBytes := []byte(fmt.Sprint(salt, pw))
// 	hash := sha256.New()
// 	for i := 0; i < numStretching; i++ {
// 		asBytes = hash.Sum(asBytes)
// 	}
// 	hashed = fmt.Sprintf("%x", asBytes)

// 	return
// }

func ProcessPassword(pw string) (hashed string, err error) {
	var bytes []byte
	bytes, err = bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost+2)
	if err != nil {
		return
	}
	hashed = string(bytes)
	return
}

func CheckPassword(pw, hashed string) (err error) {
	err = bcrypt.CompareHashAndPassword([]byte(hashed), []byte(pw))
	return
}

func GenerateRandomString(length uint) (str string, err error) {
	var i uint
	maxEx := int64(len(runeSource))
	runePool := []rune(runeSource)
	for i = 0; i < length; i++ {
		bigN, err := rand.Int(rand.Reader, big.NewInt(maxEx))
		if err != nil {
			break
		}
		n := bigN.Uint64()
		str = fmt.Sprint(str, string(runePool[n]))
	}
	return
}

func NewUuIdString() string {
	raw := uuid.New()
	return raw.String()
}

func IsEmpty(str ...string) bool {
	for _, s := range str {
		if utf8.RuneCountInString(s) == 0 {
			return true
		}
	}
	return false
}

// rabbit response

func HandleError(
	server *rabbitrpc.RabbitClient,
	logger *log.Logger,
	loggerErrorMsg string,
	corrId string,
) {
	LogError(logger).Println(loggerErrorMsg)
	SendError(
		server,
		&rabbitrpc.RabbitRPCError{
			What: "internal server error",
		},
		corrId,
	)
}

func SendError(
	server *rabbitrpc.RabbitClient,
	e *rabbitrpc.RabbitRPCError,
	corrId string,
) {
	bin, err := rabbitrpc.MakeBin(
		0,
		rabbitrpc.StatusError,
		"",
		rabbitrpc.ErrorTypeName,
		e,
	)
	if err != nil {
		panic(err)
	}

	server.Publisher.Ch <- rabbitrpc.Raws{
		Body:          bin,
		CorrelationId: corrId,
	}
}

func SendOK(
	server *rabbitrpc.RabbitClient,
	dataPtr interface{},
	dataName string,
	corrId string,
) {
	bin, err := rabbitrpc.MakeBin(
		0,
		rabbitrpc.StatusOK,
		"",
		dataName,
		dataPtr,
	)
	if err != nil {
		panic(err)
	}

	server.Publisher.Ch <- rabbitrpc.Raws{
		Body:          bin,
		CorrelationId: corrId,
	}
}
