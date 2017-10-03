package token

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/dgrijalva/jwt-go"
	"github.com/guregu/dynamo"
	d "github.com/mobingilabs/mobingi-sdk-go/pkg/debug"
)

var (
	self my
)

type my struct {
	name  string
	rsa   string
	pub   []byte
	cred  interface{}
	token string
	user  string
}

func makeRsa() {
	self.rsa = os.TempDir() + "/token/rsa/"
	_, err := os.Stat(self.rsa)
	if err == nil {
		d.Info(self.rsa + " detected.")
	} else {
		d.Info(self.rsa + " not detected. mkdir" + self.rsa)
		err = os.MkdirAll(self.rsa, 0700)
		d.Info("mkdir err : ", err)
	}
	d.Info("private start")
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	d.Info("private pre")
	privDer := x509.MarshalPKCS1PrivateKey(priv)
	d.Info("private")
	if err != nil {
		log.Fatal(err)
	}
	pubkey := priv.Public()
	pubDer, err := x509.MarshalPKIXPublicKey(pubkey)
	d.Info("pub")
	if err != nil {
		log.Fatal(err)
	}

	d.Info("pem data generated")

	pemblock := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: privDer}
	pubblock := &pem.Block{Type: "RSA PUBLIC KEY", Bytes: pubDer}
	privFile, err := os.OpenFile(self.rsa+self.user+"token.pem", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	defer privFile.Close()
	pubFile, err := os.OpenFile(self.rsa+self.user+"token.pem.pub", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	defer pubFile.Close()
	if err != nil {
		log.Fatal(err)
	}
	pem.Encode(privFile, pemblock)
	pem.Encode(pubFile, pubblock)
	d.Info("token pem file generated.")
}

type tokenReq struct {
	Username string
	Passwd   string
	jwt.StandardClaims
}
type tokenGet struct {
	Key string `json:"key"`
}

func getjson(w http.ResponseWriter, r *http.Request, inputType interface{}) interface{} {
	length, _ := strconv.Atoi(r.Header.Get("Content-Length"))
	body := make([]byte, length)
	length, _ = r.Body.Read(body)
	switch inputType.(type) {
	case tokenGet:
		ret := inputType.(tokenGet)
		_ = json.Unmarshal(body[:length], &ret)
		return ret
	case tokenReq:
		ret := inputType.(tokenReq)
		_ = json.Unmarshal(body[:length], &ret)
		return ret
	}
	return nil
}

func genJWT(cred tokenReq) *jwt.Token {
	expire := time.Hour * 800
	cred.ExpiresAt = time.Now().Add(expire).Unix()
	token := jwt.NewWithClaims(jwt.GetSigningMethod("RS512"), cred)
	self.cred = cred
	return token
}

func parseTokenTxt(tokenTxt string) (*jwt.Token, error) {
	user := tokenReq{}
	key, _ := jwt.ParseRSAPublicKeyFromPEM(self.pub)
	return jwt.ParseWithClaims(tokenTxt, &user, func(tk *jwt.Token) (interface{}, error) {
		if _, ok := tk.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", tk.Header["alg"])
		}
		return key, nil
	})
}

func Settoken(w http.ResponseWriter, r *http.Request) {
	var req tokenReq
	d.Info("debug:settoken start")
	credp := getjson(w, r, req)
	d.Info("debug:getjson")
	cred := credp.(tokenReq)
	self.user = cred.Username
	d.Info("debug:get username")
	makeRsa()
	defaultPrivKey, _ := ioutil.ReadFile(self.rsa + self.user + "token.pem")
	token := genJWT(cred)
	d.Info("debug:get token data")
	self.pub, _ = ioutil.ReadFile(self.rsa + self.user + "token.pem.pub")
	// JWTに署名する
	key, _ := jwt.ParseRSAPrivateKeyFromPEM(defaultPrivKey)
	tokenTxt, err := token.SignedString(key)
	self.token = tokenTxt
	d.Info("token generated")

	if err != nil {
		log.Fatal(err)
	}
	payload := `{"key":"` + tokenTxt + `"}`
	w.Write([]byte(payload))
}

type Event struct {
	Username string `dynamo:"username"`
	Pass     string `dynamo:"password"`
	Status   string `dynamo:"status"`
}

func CheckToken(credential string, region string, token_user string, token_pass string) (bool, error) {
	cred := credentials.NewSharedCredentials("/root/.aws/credentials", credential)
	db := dynamo.New(session.New(), &aws.Config{Region: aws.String(region),
		Credentials: cred,
	})

	table := db.Table("MC_IDENTITY")
	var results []Event
	err := table.Get("username", token_user).All(&results)
	ret := false
	for _, data := range results {
		if data.Status == "deleted" {
			ret = false
			d.Info("token_ALMuser_check: status = deleted, username = ", data.Username)
			break
		} else {
			d.Info("token_ALMuser_check: status = OK, username = ", data.Username)
		}

		if token_pass == data.Pass {
			d.Info("token_ALMuser_check: sucess")
			ret = true
		}
	}

	return ret, err
}

func GetToken(r *http.Request, credential string, awsRegion string) error {
	token := strings.Split(r.Header.Get("Authorization"), " ")[1]
	d.Info("token:", token)
	return nil

	/*
		parsedToken, _ := parseTokenTxt(token)
		d.Info(parsedToken)
		claims := *parsedToken.Claims.(*tokenReq)
		token_user := claims.Username
		token_pass := fmt.Sprintf("%x", md5.Sum([]byte(claims.Passwd)))
		d.Info("token_user:", token_user)
		d.Info("token_pass:", "xxxxx")

		tf := false
		var err error = nil
		if parsedToken.Valid {
			tf = true
		} else {
			err = errors.New("your token is not valid ")
		}
		d.Info("token_check:", tf)
		tf, err = CheckToken(credential, awsRegion, token_user, token_pass)
		if tf == false {
			err = errors.Wrap(err, "your username or password is not valid ")
		}
		return err
	*/
}

//func Test() {
//	router := mux.NewRouter()
//	router.HandleFunc("/token", settoken)
//	router.HandleFunc("/login", getToken)
//	log.Fatal(http.ListenAndServe(":8080", router))
//}
