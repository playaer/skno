package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"syscall"
	"time"
	"encoding/json"
	"bytes"
	"regexp"
	"log"
	"gopkg.in/gcfg.v1"
)

const (
	PING byte = 0x00
	ACCEPT byte = 0x01
	//BACK byte = 0x02
)

type errorString struct {
	s string
}

type Connection struct {
	ProHost string
}

type Config struct {
	Connection Connection
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

	cfg Config
)

type Payment struct {
	TotalPayed float64
}

type Order struct {
	Payment Payment
}

func main() {
	err := gcfg.ReadFileInto(&cfg, "conf.gcfg")
	if err != nil {
		fmt.Println("error:", err)
	}

	go startProxyServer()

	log.Println("Initialize")
	initialize()

	go func() {
		for {
			select {
			case <-tickChan:
				if sknoConnected {
					log.Println("ping")
					err := sendDataSkno(PING, 0)
					if err != nil {
						log.Println("Ping:", err)
					}
				} else {
					log.Println("Initialize")
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
		log.Println(err)
		return err
	}
	err = closeSknoDll.Find()
	if err != nil {
		log.Println(err)
		return err
	}
	err = sendEventSknoDll.Find()
	if err != nil {
		log.Println(err)
		return err
	}
	ret, _, err := openSknoDll.Call()
	//if err != nil {
	//	log.Println(err)
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
	//	log.Println(err)
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
	//	log.Println(err)
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

func proxy(parentResp http.ResponseWriter, parentReq *http.Request) {
	client := &http.Client{}

	log.Println(parentReq.Method, parentReq.RequestURI)

	body, _ := ioutil.ReadAll(parentReq.Body)
	//log.Println(string(body))
	clientReq, err := http.NewRequest(parentReq.Method, cfg.Connection.ProHost + parentReq.RequestURI, bytes.NewBuffer(body))
	if err != nil {
		log.Println(string(body))
		os.Exit(0)
	}

	fmt.Println(parentReq.RequestURI)
	r := regexp.MustCompile(`^\/t-api\/v1\/order\/[0-9]+$`)
	isTarget := r.MatchString(parentReq.RequestURI)
	fmt.Println("isTarget: ", isTarget)
	if parentReq.Method == "POST" && isTarget {
		var o Order
		err := json.Unmarshal(body, &o)
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println(o.Payment.TotalPayed)
			if o.Payment.TotalPayed != 0 {
				val := int(o.Payment.TotalPayed * 10000)
				go sendDataSkno(ACCEPT, val)
			}
		}
	}

	if !sknoConnected {
		parentResp.WriteHeader(http.StatusForbidden)
		return
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
		log.Println(err)
		os.Exit(0)
	}
	respBody, err := ioutil.ReadAll(clientResp.Body)
	if err != nil {
		log.Println(err)
		os.Exit(0)
	}
	//log.Println(string(respBody))
	clientResp.Body.Close()
	parentResp.Header().Set("Content-Type", clientResp.Header.Get("Content-Type"))
	parentResp.Write(respBody)
}

func startProxyServer() {

	http.HandleFunc("/", proxy)
	err := http.ListenAndServe("0.0.0.0:8888", nil)
	if err != nil {
		log.Println(err)
	}

	log.Println("Server proxy started")
}
