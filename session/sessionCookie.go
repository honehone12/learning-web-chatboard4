package session

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const sessionCookieLabel = "moemoecookie"

func StoreSessionCookie(ctx *gin.Context, value string) (err error) {
	//add exp
	value = fmt.Sprintf(
		"%s||%d",
		value,
		time.Now().Add(sessionExp).Unix(),
	)
	encrypted, err := encrypt(value)
	if err != nil {
		return
	}
	// add mac value first
	bytesVal := makeMAC(encrypted)
	// separated '||'
	bytesVal = append(bytesVal, []byte("||")...)
	// add encrypted value
	bytesVal = append(bytesVal, encrypted...)

	valToStore := base64.URLEncoding.EncodeToString(bytesVal)

	if gin.IsDebugging() {
		sessionMaker.logger.Printf(
			"stored cookie [%s] %s\n",
			sessionCookieLabel,
			valToStore,
		)
	}
	ctx.SetSameSite(http.SameSiteStrictMode)
	ctx.SetCookie(
		sessionCookieLabel,
		valToStore,
		sessionExpSec,
		"/",
		"",
		sessionMaker.useSecureCookie,
		sessionMaker.useHttpOnlyCookie,
	)
	return
}

func PickupSessionCookie(ctx *gin.Context) (value string, err error) {
	rawStored, err := ctx.Cookie(sessionCookieLabel)
	if err != nil {
		return
	}
	bytesVal, err := base64.URLEncoding.DecodeString(rawStored)
	if err != nil {
		return
	}
	splited := bytes.SplitN(bytesVal, []byte("||"), 2)
	mac := splited[0]
	encrypted := splited[1]

	if !VerifyMAC(mac, encrypted) {
		err = fmt.Errorf("invalid cookie %s", rawStored)
		return
	}

	decrypted, err := decrypt(encrypted)
	if err != nil {
		return
	}
	value, unixTimeStr, ok := strings.Cut(decrypted, "||")
	if !ok {
		err = errors.New("separator not found")
		return
	}
	unixTime, err := strconv.ParseInt(unixTimeStr, 10, 64)
	if err != nil {
		return
	}

	if unixTime < time.Now().Unix() {
		err = errors.New("session expired")
	}
	return
}
