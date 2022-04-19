package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	"html/template"
	"learning-web-chatboard4/common"
	"learning-web-chatboard4/common/models"
	"learning-web-chatboard4/rabbitrpc"
	"learning-web-chatboard4/session"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
)

const (
	publicNavbar template.HTML = `
<div class="navbar navbar-expand-md navbar-dark fixed-top bg-dark" role="navigation">
  <div class="container">
    <div class="navbar-header">
      <a class="navbar-brand" href="/">KEIJIBAN</a>
    </div>
    <div class="nav navbar-nav navbar-right">
      <a href="/user/login">Login</a>
    </div>
  </div>
</div>`

	privateNavbar template.HTML = `
<div class="navbar navbar-expand-md navbar-dark fixed-top bg-dark" role="navigation">
  <div class="container">
    <div class="navbar-header">
	  <a class="navbar-brand" href="/">KEIJIBAN</a>
    </div>
    <div class="nav navbar-nav navbar-right">
	  <a href="/user/logout">Logout</a>
    </div>
  </div>
</div>`

	replyForm template.HTML = `
<div class="panel panel-info">
  <div class="panel-body">
    <form id="post" role="form" action="/topic/post" method="post">
	  <div class="form-group">
	  	<textarea class="form-control" name="body" id="body" placeholder="Write your reply here" rows="3"></textarea>
	    <br>
		<button class="btn btn-primary pull-right" type="submit">Reply</button>
	  </div>
    </form>
  </div>
</div>`
)

const (
	minNameLen  = 1
	maxNameLen  = 100
	minEmailLen = 1
	maxEmailLen = 100
	minPwLen    = 6
	maxPwLen    = 60
	maxTopicLen = 5000
	maxReplyLen = 5000
)

func handleErrorInternal(
	loggerErrorMsg string,
	ctx *gin.Context,
	redirect bool,
) {
	common.LogError(logger).Println("recieved error:", loggerErrorMsg)
	if redirect {
		errorRedirect(ctx, "internal error")
	}
}

func getHTMLElemntInternal(isLoggedin bool) (template.HTML, template.HTML) {
	if isLoggedin {
		return privateNavbar, replyForm
	} else {
		return publicNavbar, ""
	}
}

func indexGet(ctx *gin.Context) {
	topics, err := indexGetInternal(ctx)
	if err != nil {
		handleErrorInternal(err.Error(), ctx, true)
	}
	navbar, _ := getHTMLElemntInternal(confirmLoggedIn(ctx))
	ctx.HTML(
		http.StatusOK,
		"index.html",
		gin.H{
			"navbar": navbar,
			"topics": topics,
		},
	)
}

func indexGetInternal(ctx *gin.Context) (topics []models.Topic, err error) {
	err = sendRequestAndWait(
		topicsClient,
		"readTopics",
		"Topic",
		&models.Topic{},
		func(raws rabbitrpc.Raws) (e error) {
			e = extract(&raws, &topics)
			if e != nil {
				handleErrorInternal(e.Error(), ctx, false)
			}
			return
		},
	)
	return
}

func errorRedirect(ctx *gin.Context, msg string) {
	ctx.Redirect(
		http.StatusFound,
		fmt.Sprintf(
			"%s%s",
			"/error?msg=",
			msg,
		),
	)
}

func errorGet(ctx *gin.Context) {
	errMsg := ctx.Query("msg")
	err := validate.Var(errMsg, "lowercase")
	if err != nil {
		errMsg = "internal error"
	}
	navbar, _ := getHTMLElemntInternal(confirmLoggedIn(ctx))
	ctx.HTML(
		http.StatusFound,
		"error.html",
		gin.H{
			"navbar": navbar,
			"msg":    errMsg,
		},
	)
}

func loginGet(ctx *gin.Context) {
	state := getStateFromCTX(ctx)
	ctx.HTML(
		http.StatusOK,
		"login.html",
		gin.H{
			"state": state,
		},
	)
}

func signupGet(ctx *gin.Context) {
	state := getStateFromCTX(ctx)
	ctx.HTML(
		http.StatusOK,
		"signup.html",
		gin.H{
			"state": state,
		},
	)
}

// use one more page and use post method
// or use form button
// for protecting from CSRF attack
func logoutGet(ctx *gin.Context) {
	if confirmLoggedIn(ctx) {
		err := logoutGetInternal(ctx)
		if err != nil {
			handleErrorInternal(err.Error(), ctx, true)
			return
		}
	}
	ctx.Redirect(http.StatusMovedPermanently, "/")
}

func logoutGetInternal(ctx *gin.Context) (err error) {
	sess, err := getSessionPtrFromCTX(ctx)
	if err != nil {
		return
	}

	session.DelFromRedis(sess.UuId)
	return
}

func signupPost(ctx *gin.Context) {
	err := signupPostInternal(ctx)
	if err != nil {
		handleErrorInternal(err.Error(), ctx, true)
		return
	}
	ctx.Redirect(http.StatusMovedPermanently, "/user/login")
}

func signupPostInternal(ctx *gin.Context) (err error) {
	_, err = stateCheckProcess(ctx)
	if err != nil {
		return
	}

	email := ctx.PostForm("email")
	emailLen := utf8.RuneCountInString(email)
	if emailLen < minEmailLen || emailLen > maxEmailLen {
		err = errors.New("invalid input")
	}
	err = validate.Var(email, "email")
	if err != nil {
		return
	}

	pw := ctx.PostForm("password")
	pwLen := utf8.RuneCountInString(pw)
	if pwLen < minPwLen || pwLen > maxPwLen {
		err = errors.New("invalid input")
		return
	}

	name := ctx.PostForm("name")
	nameLen := utf8.RuneCountInString(name)
	if nameLen < minNameLen || nameLen > maxNameLen {
		err = errors.New("invalid input")
	}

	// check name pw email is not same
	if strings.Compare(pw, name) == 0 ||
		strings.Compare(name, email) == 0 ||
		strings.Compare(pw, email) == 0 {

		err = errors.New("invalid input")
		return
	}

	newUser := models.User{
		Name:     name,
		Email:    email,
		Password: pw,
	}

	err = sendRequest(
		usersClient,
		"createUser",
		"User",
		&newUser,
		func(raws rabbitrpc.Raws) {
			user := models.User{}
			e := extract(&raws, &user)
			if e != nil {
				handleErrorInternal(e.Error(), ctx, false)
			}
		},
	)
	return
}

func authenticatePost(ctx *gin.Context) {
	err := authenticatePostInternal(ctx)
	if err != nil {
		handleErrorInternal(err.Error(), ctx, true)
		return
	}
	ctx.Redirect(http.StatusMovedPermanently, "/")
}

func authenticatePostInternal(ctx *gin.Context) (err error) {
	var sess *models.Session
	sess, err = stateCheckProcess(ctx)
	if err != nil {
		return
	}

	// authenticate
	email := ctx.PostForm("email")
	if utf8.RuneCountInString(email) > maxEmailLen {
		err = errors.New("invalid input")
		return
	}
	err = validate.Var(email, "email")
	if err != nil {
		return
	}

	pw := ctx.PostForm("password")
	if utf8.RuneCountInString(pw) > maxPwLen {
		err = errors.New("invalid input")
		return
	}

	authUser := models.User{
		Email:    email,
		Password: pw,
	}
	err = sendRequestAndWait(
		usersClient,
		"readUser",
		"User",
		&authUser,
		func(raws rabbitrpc.Raws) (e error) {
			e = extract(&raws, &authUser)
			if e != nil {
				handleErrorInternal(e.Error(), ctx, false)
			}
			return
		},
	)
	if err != nil {
		return
	}

	// delete session from redis to update uuid
	session.DelFromRedis(sess.UuId)

	// start new session
	sess.UuId = common.NewUuIdString()
	sess.Token = authUser.Token
	sess.UserName = authUser.Name
	sess.UserId = authUser.Id
	sess.UserEmail = authUser.Email

	session.SetToRedisWithExpiration(sess)

	err = session.StoreSessionCookie(ctx, sess.UuId)
	return
}

func topicGet(ctx *gin.Context) {
	topic, replies, err := topicGetInternal(ctx)
	if err != nil {
		handleErrorInternal(err.Error(), ctx, true)
		return
	}

	navbar, replyForm := getHTMLElemntInternal(confirmLoggedIn(ctx))
	state := getStateFromCTX(ctx)

	ctx.HTML(
		http.StatusOK,
		"topic.html",
		gin.H{
			"navbar":    navbar,
			"topic":     topic,
			"replyForm": replyForm,
			"replies":   replies,
			"state":     state,
		},
	)
}

func topicGetInternal(ctx *gin.Context,
) (topic *models.Topic, replies []models.Reply, err error) {
	base64_uuid := ctx.Query("id")
	bytes, err := base64.URLEncoding.DecodeString(base64_uuid)
	if err != nil {
		return
	}

	uuid := string(bytes)
	err = validate.Var(uuid, "uuid4")
	if err != nil {
		return
	}

	topic = &models.Topic{UuId: uuid}
	err = sendRequestAndWait(
		topicsClient,
		"readATopic",
		"Topic",
		topic,
		func(raws rabbitrpc.Raws) (e error) {
			e = extract(&raws, topic)
			if e != nil {
				handleErrorInternal(e.Error(), ctx, false)
			}
			return
		},
	)
	if err != nil {
		return
	}

	err = sendRequestAndWait(
		topicsClient,
		"readRepliesInTopic",
		"Topic",
		topic,
		func(raws rabbitrpc.Raws) (e error) {
			e = extract(&raws, &replies)
			if e != nil {
				handleErrorInternal(e.Error(), ctx, false)
			}
			return
		},
	)
	if err != nil {
		return
	}

	// store info into session
	sess, err := getSessionPtrFromCTX(ctx)
	if err != nil {
		return
	}
	sess.TopicId = topic.Id
	sess.TopicUuId = topic.UuId
	err = session.SetToRedis(sess)
	return
}

func newTopicGet(ctx *gin.Context) {
	loggedin := confirmLoggedIn(ctx)
	navbar, _ := getHTMLElemntInternal(loggedin)
	state := getStateFromCTX(ctx)
	if loggedin {
		ctx.HTML(
			http.StatusOK,
			"newtopic.html",
			gin.H{
				"navbar": navbar,
				"state":  state,
			},
		)
	} else {
		ctx.Redirect(http.StatusFound, "/user/login")
	}
}

func newTopicPost(ctx *gin.Context) {
	if !confirmLoggedIn(ctx) {
		ctx.Redirect(http.StatusFound, "/user/login")
		return
	}

	err := newTopicPostInternal(ctx)
	if err != nil {
		handleErrorInternal(err.Error(), ctx, true)
		return
	}

	ctx.Redirect(http.StatusMovedPermanently, "/")
}

func newTopicPostInternal(ctx *gin.Context) (err error) {
	sess, err := stateCheckProcess(ctx)
	if err != nil {
		return
	}

	body := ctx.PostForm("topic")
	if utf8.RuneCountInString(body) > maxTopicLen {
		err = errors.New("invalid input")
		return
	}

	topic := models.Topic{
		Topic:  body,
		Owner:  sess.UserName,
		UserId: sess.UserId,
	}
	err = sendRequestAndWait(
		topicsClient,
		"createTopic",
		"Topic",
		&topic,
		func(raws rabbitrpc.Raws) (e error) {
			e = extract(&raws, &topic)
			if e != nil {
				handleErrorInternal(e.Error(), ctx, false)
			}
			return
		},
	)
	return
}

func newReplyPost(ctx *gin.Context) {
	if !confirmLoggedIn(ctx) {
		ctx.Redirect(http.StatusFound, "/user/login")
		return
	}

	topiUuId, err := newReplyPostInternal(ctx)
	if err != nil {
		handleErrorInternal(err.Error(), ctx, true)
		return
	}
	encoded := base64.URLEncoding.EncodeToString([]byte(topiUuId))
	ctx.Redirect(http.StatusMovedPermanently, fmt.Sprint("/topic/read?id=", encoded))
}

func newReplyPostInternal(ctx *gin.Context) (topiUuId string, err error) {
	sess, err := stateCheckProcess(ctx)
	if err != nil {
		return
	}

	// pick up info from session
	topiId := sess.TopicId
	topiUuId = sess.TopicUuId

	body := ctx.PostForm("body")
	if utf8.RuneCountInString(body) > maxReplyLen {
		err = errors.New("invlid input")
		return
	}

	reply := models.Reply{
		Body:        body,
		Contributor: sess.UserName,
		UserId:      sess.UserId,
		TopicId:     topiId,
	}
	err = sendRequest(
		topicsClient,
		"createReply",
		"Reply",
		&reply,
		func(raws rabbitrpc.Raws) {
			e := extract(&raws, &reply)
			if e != nil {
				handleErrorInternal(e.Error(), ctx, false)
			}
		},
	)
	if err != nil {
		return
	}

	topic := models.Topic{UuId: topiUuId}
	err = sendRequest(
		topicsClient,
		"incrementTopic",
		"Topic",
		&topic,
		func(raws rabbitrpc.Raws) {
			e := extract(&raws, &topic)
			if e != nil {
				handleErrorInternal(e.Error(), ctx, false)
			}
		},
	)
	return
}
