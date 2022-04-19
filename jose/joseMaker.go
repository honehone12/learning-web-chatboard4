package jose

import (
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"learning-web-chatboard4/common"
	"log"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/exp/slices"
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
)

type Scope struct {
	Scopes []string
}

const (
	rsaKeySize        = 2048
	hs256KeySize uint = 64
	jwtExp            = time.Hour * 24
)

var joseMaker struct {
	isInitialized bool
	logger        *log.Logger

	privateKey   *rsa.PrivateKey
	publicKey    *rsa.PublicKey
	signatureKey []byte

	encrypter jose.Encrypter
	singner   jose.Signer

	issuer         string
	knownAudiences jwt.Audience
}

func StartJoseMaker(issuer string) (err error) {
	if joseMaker.isInitialized {
		return
	}

	joseMaker.isInitialized = true

	joseMaker.logger = log.New(
		os.Stdout,
		"[JOSE] ",
		log.Ldate|log.Ltime|log.Lshortfile,
	)

	joseMaker.issuer = issuer

	joseMaker.privateKey, err = rsa.GenerateKey(rand.Reader, rsaKeySize)
	if err != nil {
		return
	}
	joseMaker.publicKey = &joseMaker.privateKey.PublicKey

	// use rsa for future development
	joseMaker.encrypter, err = jose.NewEncrypter(
		jose.A256GCM,
		jose.Recipient{
			Algorithm: jose.RSA_OAEP_256,
			Key:       joseMaker.publicKey,
		},
		(&jose.EncrypterOptions{}).WithType("JWT").WithContentType("JWT"),
	)
	if err != nil {
		return
	}

	var sKey string
	sKey, err = common.GenerateRandomString(hs256KeySize)
	if err != nil {
		return
	}
	joseMaker.signatureKey = []byte(sKey)

	joseMaker.singner, err = jose.NewSigner(
		jose.SigningKey{
			Algorithm: jose.HS256,
			Key:       joseMaker.signatureKey,
		},
		(&jose.SignerOptions{}).WithType("JWT"),
	)
	if err != nil {
		return
	}

	return
}

func AddKnownAudience(audience string) {
	joseMaker.knownAudiences = append(joseMaker.knownAudiences, audience)
}

func NewClaims(subject, audience string) (clms *jwt.Claims, err error) {
	if !slices.Contains[string](joseMaker.knownAudiences, audience) {
		err = errors.New("unknown audience")
		return
	}

	t := time.Now()
	now := jwt.NewNumericDate(t)
	exp := jwt.NewNumericDate(t.Add(jwtExp))
	clms = &jwt.Claims{
		Issuer:    joseMaker.issuer,
		Subject:   subject,
		Audience:  jwt.Audience{audience},
		Expiry:    exp,
		NotBefore: now,
		IssuedAt:  now,
		ID:        uuid.New().String(),
	}
	return
}

func NewScope() (scp *Scope) {
	scp = &Scope{
		Scopes: []string{"read", "write"},
	}
	return
}

func MakeJWT(clms *jwt.Claims, scp *Scope) (rawToken string, err error) {
	rawToken, err = jwt.SignedAndEncrypted(
		joseMaker.singner,
		joseMaker.encrypter,
	).
		Claims(clms).
		Claims(scp).
		CompactSerialize()
	return
}

func VerifyJWT(raw, email, client string) (err error) {
	parsed, err := jwt.ParseSignedAndEncrypted(raw)
	if err != nil {
		return
	}
	decr, err := parsed.Decrypt(joseMaker.privateKey)
	if err != nil {
		return
	}

	clm := jwt.Claims{}
	scp := Scope{}
	err = decr.Claims(joseMaker.signatureKey, &clm)
	if err != nil {
		return
	}
	err = decr.Claims(joseMaker.signatureKey, &scp)
	if err != nil {
		return
	}

	// check scope has both read and write
	if !slices.Contains[string](scp.Scopes, "read") ||
		!slices.Contains[string](scp.Scopes, "write") {
		err = errors.New("out of scopes")
		return
	}

	// check exp
	now := jwt.NewNumericDate(time.Now())
	if *clm.Expiry < *now || *clm.NotBefore > *now {
		err = errors.New("token out of date")
		return
	}

	// check who issued
	if strings.Compare(clm.Issuer, joseMaker.issuer) != 0 {
		err = errors.New("unknown issuer")
		joseMaker.logger.Printf(
			"unknown issuer: clm.Issuer %s joseMaker.issuer %s\n",
			clm.Issuer,
			joseMaker.issuer,
		)
		return
	}

	// check audience
	for _, a := range clm.Audience {
		if !slices.Contains[string](joseMaker.knownAudiences, a) {
			err = errors.New("unknown audience")
			joseMaker.logger.Printf(
				"unknown audience: %s\n",
				a,
			)
			return
		}
	}

	// check
	if strings.Compare(clm.Subject, email) != 0 {
		err = errors.New("unknown subject")
		joseMaker.logger.Printf(
			"unknown subject: clm.Subject %s email %s\n",
			clm.Subject,
			email,
		)
		return
	}

	return
}
