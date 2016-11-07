all:
	$(RM) webcrawler
	export GOPATH=${PWD}
	go build -o webcrawler main.go

clean:
	$(RM) webcrawler
