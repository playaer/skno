package main

import (
	"fmt"
	"syscall"
	"net/http"
	"io"
	"io/ioutil"
	"os"
)

const (
	PING byte = 0x00
	ACCEPT byte = 0x01
	BACK byte = 0x02
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
)

func main() {
	err := initialize()
	if err != nil {
		fmt.Println("Init:", err)
		// block
	}

	fmt.Println("ping")
	err = sendDataSkno(PING, 0)
	if err != nil {
		fmt.Println("Ping:", err)
		// block
	}
	fmt.Println("accept")
	err = sendDataSkno(ACCEPT, 12000)
	if err != nil {
		fmt.Println("Accept:", err)
		// block
	}
	fmt.Println("back")
	err = sendDataSkno(BACK, 11000)
	if err != nil {
		fmt.Println("Back:", err)
		// block
	}

	fmt.Println("close")
	err = closeSkno()
	if err != nil {
		fmt.Println("Close:", err)
		// block
	}
	//go startProxyServer()
	//
	//testServer()

	//testClient()

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
	if err != nil {
		fmt.Println(err)
	}
	if ret != 1 {
		str := fmt.Sprintf("Returned %d", ret)
		return &errorString{str}
	}

	fmt.Println(ret)

	return nil
}

func closeSkno() error {
	ret, _, err := closeSknoDll.Call()
	if err != nil {
		fmt.Println(err)
	}
	if ret != 1 {
		str := fmt.Sprintf("Returned %d", ret)
		return &errorString{str}
	}

	return nil
}

func sendDataSkno(eventType byte, value int) error {
	ret, _, err := sendEventSknoDll.Call(
		uintptr(eventType),
		uintptr(value),
	)
	if err != nil {
		fmt.Println(err)
	}
	if ret != 1 {
		str := fmt.Sprintf("Returned %d", ret)
		return &errorString{str}
	}
	fmt.Println(ret)

	return nil
}

func proxy(parentResp http.ResponseWriter, parentReq *http.Request) {
	client := &http.Client{}

	fmt.Println(parentReq.Method)
	fmt.Println(parentReq.RequestURI)

	clientReq, err := http.NewRequest(parentReq.Method, "http://daroo.by" + parentReq.RequestURI, parentReq.Body)
	if err != nil {
		fmt.Println(err)
		os.Exit(0)
	}

	// set headers
	h1 := parentReq.Header.Get("name")
	clientReq.Header.Add("name", h1)

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
	clientResp.Body.Close()
	parentResp.Header().Set("Content-Type", clientResp.Header.Get("Content-Type"))
	parentResp.Write(respBody)
}

func startProxyServer() {

	http.HandleFunc("/", proxy)
	err := http.ListenAndServe(":8888", nil)
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
