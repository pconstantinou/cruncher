
build:
	go build *.go

verify: build
	go test -coverprofile=cover.out
	golint

coverage: verify
	go tool cover -html=cover.out
	

all: verify
	go install

rebase:
	git fetch
	git rebase

commit: verify
	git commit -a

push: rebase verify
	git push


check:	rebase coverage
	git status