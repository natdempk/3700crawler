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

// csrfToken is the token cookie/url param used for authentication
var csrfToken string

// sessionCookie is the cookie returned on successful login used for authentication
var sessionCookie string

// visited is a map of URLs we have already visited
var visited = make(map[string]bool)

// visitedMutex mutex so we can safely update our URLs we have visited
var visitedMutex = &sync.Mutex{}

// numThreads specifies number of requests to allow at a single time, so we don't overload the server
const numThreads = 20

// sempahore allows us to manage number of concurrent requests
var semaphore = make(chan bool, numThreads)

// flags is a channel to put flags we've found in, so we can determine when we're done
var flags = make(chan string, 5)

// safely record that we've visited a URL
func setVisited(s string) {
	visitedMutex.Lock()
	visited[s] = true
	visitedMutex.Unlock()
	return
}

// create a new connection to the fakebook server
func getConn() (conn net.Conn) {
	conn, _ = net.Dial("tcp", "fring.ccs.neu.edu:80")
	return
}

// parse the response code of the HTTP response header
func responseCode(response string) (code int) {
	stringCode := regexp.MustCompile("HTTP/1.1 ([0-9][0-9][0-9])")
	code, _ = strconv.Atoi(stringCode.FindStringSubmatch(response)[1])
	return
}

// make a GET request to a given path on fakebook
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

// login to Fakebook and get our csrf token and session id values
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

// get every HTML link out of an HTTP response
func getLinks(response string) (links []string) {
	linkRegex := regexp.MustCompile(`<a href="(.*?)">`)
	matches := linkRegex.FindAllStringSubmatch(response, -1)

	for _, match := range matches {
		links = append(links, match[1])
	}

	return
}

// goroutine to visit a page and recursively spin off more goroutines to visit unvisited links
func visitPageT(url string) {
	_ = <-semaphore                      // grab one of the semaphore keys
	defer func() { semaphore <- true }() // make sure we put it back when done
	// get page and response code
	resp := get(url)
	rc := responseCode(resp)

	// handle 301 redirects and 500 (retries)
	switch rc {
	case 301: // follow redirect
		locationRegex := regexp.MustCompile(`Location: (.*)`)
		location := locationRegex.FindStringSubmatch(resp)[1]

		go visitPageT(location)
		return
	case 500: // retry current URL
		go visitPageT(url)
		return
	}

	// check for a flag in the page body, print it and put in the flag channel if found
	flagRegex := regexp.MustCompile(`<h[1-6] class='secret_flag' style="color:red">FLAG: (.{64})</h[1-6]>`)
	maybeFlag := flagRegex.FindAllStringSubmatch(resp, -1)
	if len(maybeFlag) != 0 {
		flag := maybeFlag[0][1]
		flags <- flag
		fmt.Println(flag)
	}

	links := getLinks(resp)

	// visit all links on the page we haven't already visited
	for _, link := range links {
		if ok, _ := visited[link]; !ok {
			setVisited(link)
			go visitPageT(link)
		}
	}
}

func main() {
	username := os.Args[1]
	password := os.Args[2]

	// login
	csrfToken, sessionCookie = login(username, password)

	// initialize semaphore keys
	for i := 0; i < numThreads; i++ {
		semaphore <- true
	}

	// visit starting page and recursively visit unvisited links
	visitPageT("/fakebook/")

	// keep running until we have all 5 flags
	for len(flags) < 5 {
		time.Sleep(1000)
	}
}
