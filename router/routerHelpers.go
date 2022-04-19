package main

import (
	"bytes"
	"encoding/base64"
	"errors"
	"learning-web-chatboard4/common"
	"learning-web-chatboard4/common/models"
	"learning-web-chatboard4/rabbitrpc"
	"learning-web-chatboard4/session"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func sendRequest(
	client *rabbitrpc.RabbitClient,
	functionToCall string,
	dataTypeName string,
	dataPtr interface{},
	callback func(raws rabbitrpc.Raws),
) error {
	corrId := client.GenerateCorrelationID()
	callbackPool[corrId] = func(raws rabbitrpc.Raws) {
		callback(raws)
		doneCh <- raws.CorrelationId
	}
	bin, err := rabbitrpc.MakeBin(
		0,
		0,
		functionToCall,
		dataTypeName,
		dataPtr,
	)
	if err != nil {
		return err
	}

	client.Publisher.Ch <- rabbitrpc.Raws{
		Body:          bin,
		CorrelationId: corrId,
	}
	return nil
}

func sendRequestAndWait(
	client *rabbitrpc.RabbitClient,
	functionToCall string,
	dataTypeName string,
	dataPtr interface{},
	callback func(raws rabbitrpc.Raws) error,
) error {
	wait := make(chan error)
	go func() {
		err := sendRequest(
			client,
			functionToCall,
			dataTypeName,
			dataPtr,
			func(raws rabbitrpc.Raws) {
				e := callback(raws)
				wait <- e
			},
		)
		if err != nil {
			wait <- err
		}
	}()
	select {
	case e := <-wait:
		return e
	}
}

func extract(raws *rabbitrpc.Raws, dataPtr interface{}) error {
	envelop, e := rabbitrpc.FromBin(raws.Body)
	if e != nil {
		return errors.New(e.What)
	}

	if envelop.Status == rabbitrpc.StatusError {
		common.LogWarning(logger).Println("returned status is error")
		rerr := &rabbitrpc.RabbitRPCError{}
		e = envelop.Extract(rerr)
		if e != nil {
			return errors.New(e.What)
		}
		return errors.New(rerr.What)
	}

	e = envelop.Extract(dataPtr)
	if e != nil {
		return errors.New(e.What)
	}
	return nil
}

func stateCheckProcess(ctx *gin.Context) (sess *models.Session, err error) {
	sess, err = getSessionPtrFromCTX(ctx)
	if err != nil {
		return
	}

	// check state
	state := ctx.PostForm("state")
	err = checkState(state, sess.State)
	if err != nil {
		return
	}

	// state is consumed, delete it
	sess.State = ""
	err = session.SetToRedis(sess)
	return
}

func checkState(exposedVal, privateVal string) (err error) {
	if strings.Compare(exposedVal, "") == 0 {
		err = errors.New("exposed value is empty")
		return
	}
	if strings.Compare(privateVal, "") == 0 {
		err = errors.New("private value is empty")
		return
	}

	bytesVal, err := base64.URLEncoding.DecodeString(exposedVal)
	if err != nil {
		return
	}
	splited := bytes.SplitN(bytesVal, []byte("||"), 2)
	// mac can store any bytes,
	// this should be URL encoded until validation
	macStored := splited[0]
	stateStored := string(splited[1])

	if !session.VerifyMAC(macStored, []byte(privateVal)) {
		err = errors.New("invalid mac")
		return
	}
	if strings.Compare(stateStored, privateVal) != 0 {
		err = errors.New("invalid state")
		return
	}
	_, unixTimeStr, ok := strings.Cut(stateStored, "||")
	if !ok {
		err = errors.New("separator not found")
		return
	}
	unixTime, err := strconv.ParseInt(unixTimeStr, 10, 64)
	if err != nil {
		return
	}
	if unixTime < time.Now().Unix() {
		err = errors.New("state expired")
	}
	return
}
