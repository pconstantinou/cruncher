
build:
	go build *.go

verify: build
	go test -coverprofile=cover.out
	go tool cover -html=cover.out
	golint

all: verify
	go install

rebase:
	git fetch
	git rebase

commit: verify
	git commit -a

push: rebase verify
	git push