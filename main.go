package main

import (
	"bufio"
	"fmt"
	"net"
	"regexp"
	"time"
)

// One post is actually made
// So only specify

// Request type, http version, user-agent, host, accept etc
// cookies

// Parse responses anyway

func main() {

	conn, _ := net.DialTimeout("tcp", "fring.ccs.neu.edu:80", time.Second)

	getReq := `GET /accounts/login/?next=/fakebook HTTP/1.1
User-Agent: curl/7.40.0
Host: fring.ccs.neu.edu
Content-Length: 0
Accept: */*

`
	fmt.Println(time.Now())
	fmt.Fprint(conn, getReq)
	readerB := bufio.NewReader(conn)
	buf := make([]byte, 10000)

	fmt.Println(time.Now())
	readerB.Read(buf)
	resp := string(buf)
	fmt.Println(resp)

	conn.Close()

	fmt.Println(time.Now())
	csrf := regexp.MustCompile("csrfmiddlewaretoken' value='([a-f0-9]*)'")
	sessionCookie := regexp.MustCompile("Set-Cookie: sessionid=([a-f0-9]*)")
	csrfToken := csrf.FindStringSubmatch(resp)[1]
	sessCoo := sessionCookie.FindStringSubmatch(resp)[1]

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

username=001178291&password=418VQU32&csrfmiddlewaretoken=%s&next=%%2Ffakebook%%2F

`

	fmt.Println(time.Now())

	conn, _ = net.DialTimeout("tcp", "fring.ccs.neu.edu:80", time.Second)

	fmt.Println(time.Now())
	fmt.Fprintf(conn, postReq, csrfToken, sessCoo, csrfToken)

	reader := bufio.NewScanner(conn)
	resp = ""

	fmt.Println(time.Now())
	for reader.Scan() {
		resp += reader.Text() + "\n"
	}

	fmt.Println(time.Now())
	sessCoo = sessionCookie.FindStringSubmatch(resp)[1]

	conn.Close()

	fmt.Println(time.Now())
	getReq = `GET /fakebook/ HTTP/1.1
User-Agent: curl/7.40.0
Host: fring.ccs.neu.edu
Cookie: csrftoken=%s; sessionid=%s
Accept: */*

`

	conn, _ = net.DialTimeout("tcp", "fring.ccs.neu.edu:80", time.Second)
	fmt.Fprintf(conn, getReq, csrfToken, sessCoo)

	reader = bufio.NewScanner(conn)
	resp = ""
	for reader.Scan() {
		resp += reader.Text() + "\n"
	}
	fmt.Println(resp)
}
