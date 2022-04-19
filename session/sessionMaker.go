package session

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"learning-web-chatboard4/common"
	"learning-web-chatboard4/common/models"
	"log"
	"os"
	"time"

	"github.com/gomodule/redigo/redis"
)

const (
	aes256ENCKeySize uint = 32
	sha256MACKeySize uint = 64
	stateSize        uint = 32
	stateExp              = time.Minute * 5
	sessionExp            = time.Hour
	sessionExpSec         = int(sessionExp / time.Second)
)

var sessionMaker struct {
	isInitialized     bool
	logger            *log.Logger
	redisConn         redis.Conn
	block             cipher.Block
	macKey            []byte
	useSecureCookie   bool
	useHttpOnlyCookie bool
}

// every time server is restarted, cookie become no longer valid
func StartSessionMaker(useSecure, useHttpOnly bool) (err error) {
	if sessionMaker.isInitialized {
		return
	}

	sessionMaker.isInitialized = true

	sessionMaker.redisConn, err = redis.Dial(
		redisConnectionKind,
		redisAddress,
	)
	if err != nil {
		return
	}

	bKyeStr, err := common.GenerateRandomString(aes256ENCKeySize)
	if err != nil {
		return
	}
	bKey := []byte(bKyeStr)
	macKeyStr, err := common.GenerateRandomString(sha256MACKeySize)
	if err != nil {
		return
	}
	sessionMaker.macKey = []byte(macKeyStr)
	sessionMaker.block, err = aes.NewCipher(bKey)

	sessionMaker.logger = log.New(
		os.Stdout,
		"[SESSION] ",
		log.Ldate|log.Ltime|log.Lshortfile,
	)

	sessionMaker.useSecureCookie = useSecure
	sessionMaker.useHttpOnlyCookie = useHttpOnly
	return
}

func NewSession() (sess *models.Session) {
	sess = &models.Session{
		UuId:      common.NewUuIdString(),
		CreatedAt: time.Now(),
	}
	return
}

func GenerateState() (stateRaw, stateAndMACEncoded string, err error) {
	state, err := common.GenerateRandomString(stateSize)
	if err != nil {
		return
	}
	state = fmt.Sprintf(
		"%s||%d",
		state,
		time.Now().Add(stateExp).Unix(),
	)
	stateRaw = state

	// same proc with session cookie
	stateAsBytes := []byte(state)
	bytesVal := makeMAC(stateAsBytes)
	bytesVal = append(bytesVal, []byte("||")...)
	bytesVal = append(bytesVal, stateAsBytes...)

	stateAndMACEncoded = base64.URLEncoding.EncodeToString(bytesVal)
	return
}

func VerifyMAC(mac, value []byte) bool {
	hashedVal := makeMAC(value)
	return hmac.Equal(mac, hashedVal)
}

func makeMAC(value []byte) []byte {
	// possible to cache??
	hash := hmac.New(sha256.New, sessionMaker.macKey)
	hash.Write(value)
	return hash.Sum(nil)
}

func encrypt(plainText string) (cipherText []byte, err error) {
	cipherText = make([]byte, aes.BlockSize+len(plainText))
	iv := cipherText[:aes.BlockSize]
	n, err := io.ReadFull(rand.Reader, iv)
	if err != nil {
		err = fmt.Errorf("%s: returned %d", err.Error(), n)
		return
	}

	encryptStream := cipher.NewCTR(sessionMaker.block, iv)
	encryptStream.XORKeyStream(cipherText[aes.BlockSize:], []byte(plainText))
	return
}

func decrypt(cipherText []byte) (plainText string, err error) {
	decryptText := make([]byte, len(cipherText[aes.BlockSize:]))
	decryptStream := cipher.NewCTR(sessionMaker.block, cipherText[:aes.BlockSize])
	decryptStream.XORKeyStream(decryptText, cipherText[aes.BlockSize:])
	plainText = string(decryptText)
	return
}
