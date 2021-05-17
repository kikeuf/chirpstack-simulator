package iapi

import (
	"encoding/json"
	"bytes"
	"io/ioutil"
   	"net/http"
	//"strings"
	//"fmt"
)

func BytesToString(data []byte) string {
	return string(data[:])
}

func GetToken(server string, user string, psw string) string {

	// Generated by curl-to-Go: https://mholt.github.io/curl-to-go

	// curl -X POST --header 'Content-Type: application/json' --header 'Accept: application/json' -d '{ \
	//    "email": "admin", \
	//    "password": "password" \
	//  }' 'http://localhost:8080/api/internal/login'

	if len(server)==0 { server = "http://localhost:8080" }
	
	type Payload struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	data := Payload{
		Email : 	user,			
		Password:       psw,
	}
	payloadBytes, err := json.Marshal(data)
	if err != nil {
		// handle err
		return ""
	}
	body := bytes.NewReader(payloadBytes)

	req, err := http.NewRequest("POST", server + "/api/internal/login", body)
	if err != nil {
		// handle err
		return ""
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		// handle err
		return ""
	}
	
	defer resp.Body.Close()
	bdy, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		// handle err
		return ""
	}
	sbdy:=bdy[8:len(bdy)-2]
	return BytesToString(sbdy)
}
