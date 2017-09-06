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
	"path/filepath"
	"strconv"

	"time"

	"github.com/dgrijalva/jwt-go"
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
	self.name, _ = os.Executable()
	self.rsa = filepath.Dir(self.name) + "/token/rsa"
	_, err := os.Stat(self.rsa)
	if err == nil {
		log.Println(self.rsa + "/token/rsa detected.")
	} else {
		log.Println(self.rsa + "/token/rsa not detected. mkdir /token/rsa")
		os.Mkdir(self.rsa, 0700)
	}
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	privDer := x509.MarshalPKCS1PrivateKey(priv)
	if err != nil {
		log.Fatal(err)
	}
	pubkey := priv.Public()
	pubDer, err := x509.MarshalPKIXPublicKey(pubkey)
	if err != nil {
		log.Fatal(err)
	}

	pemblock := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: privDer}
	pubblock := &pem.Block{Type: "RSA PUBLIC KEY", Bytes: pubDer}
	privFile, err := os.OpenFile(self.rsa+self.user+".pem", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	defer privFile.Close()
	pubFile, err := os.OpenFile(self.rsa+self.user+".pem.pub", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	defer pubFile.Close()
	if err != nil {
		log.Fatal(err)
	}
	pem.Encode(privFile, pemblock)
	pem.Encode(pubFile, pubblock)
	log.Println("token pem file generated.")
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

func genJWT(w http.ResponseWriter, r *http.Request) *jwt.Token {
	var req tokenReq
	credp := getjson(w, r, req)
	cred := credp.(tokenReq)
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
	credp := getjson(w, r, req)
	cred := credp.(tokenReq)
	self.user = cred.Username
	makeRsa()
	defaultPrivKey, _ := ioutil.ReadFile(self.rsa + self.user + ".pem")
	token := genJWT(w, r)
	self.pub, _ = ioutil.ReadFile(self.rsa + self.user + ".pem.pub")
	// JWTに署名する
	key, _ := jwt.ParseRSAPrivateKeyFromPEM(defaultPrivKey)
	tokenTxt, err := token.SignedString(key)
	self.token = tokenTxt
	log.Println("token generated")

	if err != nil {
		log.Fatal(err)
	}
	payload := `{"key":"` + tokenTxt + `"}`
	w.Write([]byte(payload))
}

func GetToken(w http.ResponseWriter, r *http.Request) (bool, string) {
	tokenjson := r.Header.Get("Token")
	tokens := tokenGet{}
	json.Unmarshal([]byte(tokenjson), &tokens)
	token := tokens.Key
	parsedToken, _ := parseTokenTxt(token)
	//claims := *parsedToken.Claims.(*tokenReq)
	payload := ""
	tf := false
	if parsedToken.Valid {
		payload = fmt.Sprint("your token is valid ", parsedToken.Valid)
		tf = true
	} else {
		payload = fmt.Sprint("your token is not valid ", parsedToken.Valid)
	}
	return tf, payload
}

//func Test() {
//	router := mux.NewRouter()
//	router.HandleFunc("/token", settoken)
//	router.HandleFunc("/login", getToken)
//	log.Fatal(http.ListenAndServe(":8080", router))
//}
