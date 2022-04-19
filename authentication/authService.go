package main

import (
	"errors"
	"fmt"
	"learning-web-chatboard4/common"
	"learning-web-chatboard4/common/models"
	"learning-web-chatboard4/jose"
	"strings"
	"time"
)

const (
	usersTable  = "users"
	loginsTable = "logins"
)

const (
	numStretching int  = 10000
	pwSaltSize    uint = 20
)

func createUser(user *models.User, corrId string) {
	err := createUserInternal(user)
	if err != nil {
		common.HandleError(server, logger, err.Error(), corrId)
		return
	}

	common.SendOK(server, user, "User", corrId)
}

func createUserInternal(user *models.User) (err error) {
	if common.IsEmpty(
		user.Name,
		user.Email,
		user.Password,
	) {
		err = errors.New("contains empty string")
		return
	}

	user.Salt, err = common.GenerateRandomString(pwSaltSize)
	if err != nil {
		return
	}
	user.Password = common.ProcessPassword(
		user.Password,
		user.Salt,
		numStretching,
	)
	user.UuId = common.NewUuIdString()
	user.CreatedAt = time.Now()
	err = createUserSQL(user)
	return
}

func createUserSQL(user *models.User) (err error) {
	affected, err := dbEngine.
		Table(usersTable).
		InsertOne(user)
	if err == nil && affected != 1 {
		err = fmt.Errorf(
			"something wrong. returned value was %d",
			affected,
		)
	}
	return
}

func readUser(user *models.User, corrId string) {
	token, err := readUserInternal(user)
	if err != nil {
		common.HandleError(server, logger, err.Error(), corrId)
		return
	}

	// not saved in database
	user.Token = token
	user.Password = ""
	user.Salt = ""

	common.SendOK(server, user, "User", corrId)
}

// actualy authentication process
func readUserInternal(user *models.User) (token string, err error) {
	if common.IsEmpty(user.Email, user.Password) {
		err = errors.New("need email and password")
		return
	}
	pw := user.Password
	user.Password = ""
	err = readUserSQL(user)
	if err != nil {
		return
	}
	pw = common.ProcessPassword(
		pw,
		user.Salt,
		numStretching,
	)
	if strings.Compare(pw, user.Password) != 0 {
		err = errors.New("password mismatch")
		return
	}

	clm, err := jose.NewClaims(user.Email, audienceName)
	if err != nil {
		return
	}
	scp := jose.NewScope()
	token, err = jose.MakeJWT(clm, scp)
	return
}

func readUserSQL(user *models.User) (err error) {
	var ok bool
	ok, err = dbEngine.
		Table(usersTable).
		Get(user)
	if err == nil && !ok {
		err = errors.New("no such user")
	}
	return
}

func verifyToken(token *common.Token, corrId string) (err error) {
	err = jose.VerifyJWT(
		token.Raw,
		token.UserEmail,
		audienceName,
	)
	if err != nil {
		common.HandleError(server, logger, err.Error(), corrId)
	}

	msg := common.SimpleMessage{
		Message: "verified",
	}
	common.SendOK(server, &msg, "SimpleMessage", corrId)
	return
}
