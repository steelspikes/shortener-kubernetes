set GOOS=linux
set GOARCH=amd64
set CGO_ENABLED=0
go build -o services/api/executable ./services/api/cmd
go build -o services/worker/executable ./services/worker/cmd