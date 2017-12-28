
all:
	go build *.go
	go test -coverprofile=cover.out
	go tool cover -html=cover.out
	go install


rebase:
	git fetch
	git rebase

commit: rebase all
	git commit -a
