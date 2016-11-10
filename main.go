package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"syscall"
	"time"
	"encoding/json"
	"bytes"
)

const (
	PING byte = 0x00
	ACCEPT byte = 0x01
	BACK byte = 0x02

	TARGET_URI string = ""
	PRO_HOST string = "http://192.168.100.54:8001"
)

type errorString struct {
	s string
}

func (e *errorString) Error() string {
	return e.s
}

var (
	skno = syscall.NewLazyDLL("skno.dll")
	openSknoDll = skno.NewProc("open")
	closeSknoDll = skno.NewProc("close")
	sendEventSknoDll = skno.NewProc("sendEvent")

	sknoConnected bool = false

	tickChan = time.NewTicker(time.Second * 55).C
)

type Payment struct {
	TotalPayed float64
}

type Order struct {
	Payment Payment
}

func main() {

	//fmt.Println("accept")
	//sendDataSkno(ACCEPT, 12000)

	go startProxyServer()
	//
	//testServer()

	//testClient()

	fmt.Println("Initialize")
	initialize()

	go func() {
		for {
			select {
			case <-tickChan:
				if sknoConnected {
					fmt.Println("ping")
					err := sendDataSkno(PING, 0)
					if err != nil {
						fmt.Println("Ping:", err)
					}
				} else {
					fmt.Println("Initialize")
					initialize()
				}
			}
		}
	}()

	var input string
	fmt.Scanln(&input)
}

func initialize() error {
	err := openSknoDll.Find()
	if err != nil {
		fmt.Println(err)
		return err
	}
	err = closeSknoDll.Find()
	if err != nil {
		fmt.Println(err)
		return err
	}
	err = sendEventSknoDll.Find()
	if err != nil {
		fmt.Println(err)
		return err
	}
	ret, _, err := openSknoDll.Call()
	//if err != nil {
	//	fmt.Println(err)
	//}
	if ret != 1 {
		str := fmt.Sprintf("Returned %d", ret)
		return &errorString{str}
	}

	sknoConnected = true

	return nil
}

func closeSkno() error {
	ret, _, _ := closeSknoDll.Call()
	//if err != nil {
	//	fmt.Println(err)
	//}
	if ret != 1 {
		str := fmt.Sprintf("Returned %d", ret)
		return &errorString{str}
	}

	return nil
}

func sendDataSkno(eventType byte, value int) error {
	ret, _, _ := sendEventSknoDll.Call(
		uintptr(eventType),
		uintptr(value),
	)
	//if err != nil {
	//	fmt.Println(err)
	//}
	if ret != 1 {
		str := fmt.Sprintf("Returned %d", ret)
		sknoConnected = false
		closeSkno()
		return &errorString{str}
	}
	sknoConnected = true

	return nil
}

func target(parentResp http.ResponseWriter, parentReq *http.Request) {
	if parentReq.Method == "POST" {
		//params := parentReq.URL.Query()
		//orderId := params.Get(":orderId")

		decoder := json.NewDecoder(parentReq.Body)
		var o Order
		err := decoder.Decode(&o)
		if err != nil {
			panic(err)
			defer parentReq.Body.Close()
		}
		fmt.Println(o.Payment.TotalPayed)
		if o.Payment.TotalPayed != 0 {
			val := int(o.Payment.TotalPayed * 10000)
			fmt.Println("accept", val)
			sendDataSkno(ACCEPT, val)
		}
	}

	proxy(parentResp, parentReq)
}

func proxy(parentResp http.ResponseWriter, parentReq *http.Request) {
	client := &http.Client{}

	fmt.Println(parentReq.Method, parentReq.RequestURI)

	body, _ := ioutil.ReadAll(parentReq.Body)
	//fmt.Println(string(body))
	clientReq, err := http.NewRequest(parentReq.Method, PRO_HOST + parentReq.RequestURI, bytes.NewBuffer(body))
	if err != nil {
		fmt.Println(string(body))
		os.Exit(0)
	}

	c := http.Cookie{
		Name: "XDEBUG_SESSION",
		Value: "terminal",
	}
	clientReq.AddCookie(&c)

	// set headers
	h1 := parentReq.Header.Get("X-Api-Token")
	h2 := parentReq.Header.Get("x-api-token")
	if h1 != "" {
		clientReq.Header.Add("X-Api-Token", h1)
	}
	if h2 != "" {
		clientReq.Header.Add("X-Api-Token", h2)
	}
	clientReq.Header.Set("Content-Type", parentReq.Header.Get("Content-Type"))

	clientResp, err := client.Do(clientReq)
	if err != nil {
		fmt.Println(err)
		os.Exit(0)
	}
	respBody, err := ioutil.ReadAll(clientResp.Body)
	if err != nil {
		fmt.Println(err)
		os.Exit(0)
	}
	//fmt.Println(string(respBody))
	clientResp.Body.Close()
	parentResp.Header().Set("Content-Type", clientResp.Header.Get("Content-Type"))
	parentResp.Write(respBody)
}

func startProxyServer() {

	http.HandleFunc("/", proxy)
	http.HandleFunc("/t-api/v1/order/:orderId", target)
	err := http.ListenAndServe("0.0.0.0:8888", nil)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println("Server proxy started")
}

func testClient() {
	fmt.Println("Client send request")
	resp, err := http.Get("http://127.0.0.1:8888")
	if err != nil {
		fmt.Println(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	fmt.Println("Client: ", string(body))
}

func testServer() {
	http.HandleFunc("/test/*", hello)
	err := http.ListenAndServe(":9999", nil)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println("Server started")
}

func hello(res http.ResponseWriter, req *http.Request) {
	res.Header().Set(
		"Content-Type",
		"text/html",
	)
	postData := ""
	if req.Method == "POST" {
		postData = req.FormValue("bbb")
	}
	io.WriteString(
		res,
		`<doctype html>
<html>
	<head>
		<title>Hello World</title>
	</head>
	<body>
		Hello World! ` + postData + `
	</body>
</html>`,
	)
}
