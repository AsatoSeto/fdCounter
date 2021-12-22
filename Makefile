build:
	CGO_ENABLED=0 GO111MODULE=on go build -o ./bin/fdcount ./main.go