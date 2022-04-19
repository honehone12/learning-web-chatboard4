package main

import (
	"errors"
	"learning-web-chatboard4/common"
	"learning-web-chatboard4/common/models"
	"learning-web-chatboard4/rabbitrpc"
	"learning-web-chatboard4/session"
	"log"

	"github.com/gin-gonic/gin"
)

const (
	loggedInLabel   = "logged-in"
	sessionPtrLabel = "session-ptr"
	stateLabel      = "state"
)

func setCommonHeadersMiddleware(ctx *gin.Context) {
	ctx.Header("Cache-Control", "no-store")
	ctx.Header("X-Frame-Options", "DENY")
	ctx.Header("X-Content-Type-Options", "nosniff")
	ctx.Next()
}

func sessionCheckMiddleware(ctx *gin.Context) {
	err := checkSessionInternal(ctx)
	if err != nil {
		if gin.IsDebugging() {
			log.Fatalln("!!MIDDLEWARE ERROR!!", err.Error())
		} else {
			log.Printf("!!MIDDLEWARE ERROR!! %s\n", err.Error())
		}
	}
	ctx.Next()
}

func checkSessionInternal(ctx *gin.Context) (err error) {
	sess := &models.Session{}
	uuid, err := session.PickupSessionCookie(ctx)
	// cookie is valid
	if err == nil {
		exists := false
		exists, err = session.ConfirmExistsFromRedis(uuid)
		if err != nil {
			return
		}
		if exists {
			// get session from redis
			err = session.GetFromRedis(uuid, sess)
			if err != nil {
				return
			}
		} else {
			err = errors.New("cookie looks valid, but redis not stored")
		}
	}
	// no cookie or cookie is invalid or expired
	if err != nil {
		if gin.IsDebugging() {
			log.Printf(
				"[MIDDLEWARE] creating new session because [%s]\n",
				err.Error(),
			)
		}
		// create new session
		sess = session.NewSession()
		err = session.SetToRedisWithExpiration(sess)
		if err != nil {
			return
		}
		err = session.StoreSessionCookie(ctx, sess.UuId)
		if err != nil {
			return
		}
	}

	ctx.Set(sessionPtrLabel, sess)
	err = nil
	return
}

func loggedInCheckMiddleware(ctx *gin.Context) {
	loggedIn, err := checkLoginInternal(ctx)
	if err != nil {
		if gin.IsDebugging() {
			log.Fatalln("!!MIDDLEWARE ERROR!!", err.Error())
		} else {
			log.Printf("!!MIDDLEWARE ERROR!! %s\n", err.Error())
		}
	}
	ctx.Set(loggedInLabel, loggedIn)
	ctx.Next()
}

func checkLoginInternal(ctx *gin.Context) (loggedIn bool, err error) {
	sess, err := getSessionPtrFromCTX(ctx)
	if err != nil {
		return
	}
	if len(sess.Token) == 0 {
		loggedIn = false
		return
	}

	token := &common.Token{
		UserEmail: sess.UserEmail,
		Raw:       sess.Token,
	}
	err = sendRequestAndWait(
		usersClient,
		"verifyToken",
		"Token",
		token,
		func(raws rabbitrpc.Raws) (e error) {
			e = extract(&raws, &common.SimpleMessage{})
			if e != nil {
				handleErrorInternal(e.Error(), ctx, false)
			}
			return
		},
	)
	if err != nil {
		loggedIn = false
		// bad design
		// can't sort unexpected error from failing verifications
		err = nil
		return
	}
	loggedIn = true
	return
}

func generateSessionStateMiddleware(ctx *gin.Context) {
	state, err := generateSessionStateInternal(ctx)
	if err != nil {
		if gin.IsDebugging() {
			log.Fatalln("!!MIDDLEWARE NOTWORKING!!", err.Error())
		} else {
			log.Printf("!!MIDDLEWARE NOTWORKING!! %s\n", err.Error())
		}
	}

	ctx.Set(stateLabel, state)
	ctx.Next()
}

func generateSessionStateInternal(ctx *gin.Context) (stateAndMACEncoded string, err error) {
	sess, err := getSessionPtrFromCTX(ctx)
	if err != nil {
		return
	}

	sess.State, stateAndMACEncoded, err = session.GenerateState()
	if err != nil {
		return
	}
	err = session.SetToRedis(sess)
	if err != nil {
		return
	}
	ctx.Set(sessionPtrLabel, sess)
	return
}
