package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"regexp"
	"strconv"
	"sync"
	"time"
)

// One post is actually made
// So only specify

// Request type, http version, user-agent, host, accept etc
// cookies

// handle errors/redirects 301 (follow), 200, 403, 404, (abandon) 500 (retry)

// globals

var csrfToken string
var sessionCookie string

var visited map[string]bool = make(map[string]bool)
var queue []string

var visitedMutex = &sync.Mutex{}

const numThreads = 20

var semaphore = make(chan bool, numThreads)
var flags = make(chan string, 5)

func setVisited(i string) {
	visitedMutex.Lock()
	visited[i] = true
	visitedMutex.Unlock()
	return

}

func getConn() (conn net.Conn) {
	conn, _ = net.Dial("tcp", "fring.ccs.neu.edu:80")
	return
}

func responseCode(response string) (code int) {
	stringCode := regexp.MustCompile("HTTP/1.1 ([0-9][0-9][0-9])")
	code, _ = strconv.Atoi(stringCode.FindStringSubmatch(response)[1])
	return
}

func get(path string) (response string) {
	conn := getConn()

	getReq := fmt.Sprintf(`GET %s HTTP/1.1
User-Agent: curl/7.40.0
Host: fring.ccs.neu.edu
Cookie: csrftoken=%s; sessionid=%s
Accept: */*

`, path, csrfToken, sessionCookie)

	fmt.Fprintf(conn, getReq)

	readerB := bufio.NewReader(conn)
	buf := make([]byte, 1000000)

	readerB.Read(buf)
	response = string(buf)

	conn.Close()
	return
}

func login(username, password string) (csrf, session string) {
	resp := get("/accounts/login/?next=/fakebook")

	csrfRegex := regexp.MustCompile("csrfmiddlewaretoken' value='([a-f0-9]*)'")
	sessionCookieRegex := regexp.MustCompile("Set-Cookie: sessionid=([a-f0-9]*)")
	csrf = csrfRegex.FindStringSubmatch(resp)[1]
	session = sessionCookieRegex.FindStringSubmatch(resp)[1]

	// lol set content length
	postReq := `POST /accounts/login/ HTTP/1.1
Host: fring.ccs.neu.edu
Origin: http://fring.ccs.neu.edu
User-Agent: curl/7.40.0
Content-Type: application/x-www-form-urlencoded
Accept: */*
Upgrade-Insecure-Requests: 1
Content-Length: 109
Referer: http://fring.ccs.neu.edu/accounts/login/?next=/fakebook/
Cookie: csrftoken=%s; sessionid=%s

username=%s&password=%s&csrfmiddlewaretoken=%s&next=%%2Ffakebook%%2F

`

	conn := getConn()

	fmt.Fprintf(conn, postReq, csrf, session, username, password, csrf)

	readerB := bufio.NewReader(conn)
	buf := make([]byte, 1000000)

	readerB.Read(buf)
	response := string(buf)

	session = sessionCookieRegex.FindStringSubmatch(response)[1]

	conn.Close()

	return
}

func getLinks(response string) (links []string) {
	linkRegex := regexp.MustCompile(`<a href="(.*?)">`)
	matches := linkRegex.FindAllStringSubmatch(response, -1)

	for _, match := range matches {
		links = append(links, match[1])
	}

	return
}

func visitPageT(url string) {
	_ = <-semaphore // beautiful
	defer func() { semaphore <- true }()
	//fmt.Printf("Visiting: %s\n", url)
	resp := get(url)
	rc := responseCode(resp)

	switch rc {
	case 301:
		locationRegex := regexp.MustCompile(`Location: (.*)`)
		location := locationRegex.FindStringSubmatch(resp)[1]
		//queue = append(queue, location)
		go visitPageT(location)
		return
	case 500:
		go visitPageT(url)
		return
	}

	flagRegex := regexp.MustCompile(`<h[1-6] class='secret_flag' style="color:red">FLAG: (.{64})</h[1-6]>`)
	maybeFlag := flagRegex.FindAllStringSubmatch(resp, -1)
	if len(maybeFlag) != 0 {
		//fmt.Println("FOUND FLAG")
		flag := maybeFlag[0][1]
		flags <- flag
		fmt.Println(flag)
	}

	links := getLinks(resp)

	for _, link := range links {
		if ok, _ := visited[link]; !ok {
			//queue = append(queue, link)
			//visited[link] = true
			setVisited(link)
			go visitPageT(link)
		}
	}
}

func main() {

	username := os.Args[1]
	password := os.Args[2]

	csrfToken, sessionCookie = login(username, password)

	// starting point
	//queue = append(queue, "/fakebook/")

	for i := 0; i < numThreads; i++ {
		semaphore <- true
	}

	visitPageT("/fakebook/")

	for len(flags) < 5 {
		time.Sleep(1000)
	}
}
