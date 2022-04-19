package main

import (
	"errors"
	"learning-web-chatboard4/common/models"
	"log"

	"github.com/gin-gonic/gin"
)

func confirmLoggedIn(ctx *gin.Context) (isLoggedIn bool) {
	val, ok := ctx.Get(loggedInLabel)
	if !ok {
		log.Println("[MIDDLEWARE] isloggedin not generated yet")
		return
	}
	if isLoggedIn, ok = val.(bool); !ok {
		if gin.IsDebugging() {
			log.Fatalln("!!MIDDLEWARE BROKEN!! isloggedin is not bool")
		} else {
			log.Println("!!MIDDLEWARE BROKEN!! isloggedin is not bool")
		}
	}
	return
}

func getSessionPtrFromCTX(ctx *gin.Context) (ptr *models.Session, err error) {
	val, ok := ctx.Get(sessionPtrLabel)
	if !ok {
		err = errors.New("session-ptr is not stored")
		return
	}
	if ptr, ok = val.(*models.Session); !ok {
		if gin.IsDebugging() {
			log.Fatalln("!!MIDDLEWARE BROKEN!! session-ptr is not *Session")
		}
		err = errors.New("!!MIDDLEWARE BROKEN!! session-ptr is not *Session")
	}
	return
}

func getStateFromCTX(ctx *gin.Context) (state string) {
	val, ok := ctx.Get(stateLabel)
	if !ok {
		log.Println("[MIDDLEWARE] state not generated yet")
		return
	}
	if state, ok = val.(string); !ok {
		if gin.IsDebugging() {
			log.Fatalln("!!MIDDLEWARE BROKEN!! state is not string")
		} else {
			log.Println("!!MIDDLEWARE BROKEN!! state is not string")
		}
	}
	return
}
