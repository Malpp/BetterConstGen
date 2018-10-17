set GOARCH=amd64
set GOOS=windows
go build .
set GOOS=linux
go build .