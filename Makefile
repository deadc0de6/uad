SRC = uad.go page.go
BIN = uad

all: build

build:
	GO111MODULE=on go build -o $(BIN) $(SRC)

build-linux:
	GO111MODULE=on GOOS=linux GOARCH=arm go build -v -o $(BIN)-linux-arm $(SRC)
	GO111MODULE=on GOOS=linux GOARCH=arm64 go build -v -o $(BIN)-linux-arm64 $(SRC)
	GO111MODULE=on GOOS=linux GOARCH=386 go build -v -o $(BIN)-linux-386 $(SRC)
	GO111MODULE=on GOOS=linux GOARCH=amd64 go build -v -o $(BIN)-linux-amd64 $(SRC)

build-windows:
	GO111MODULE=on GOOS=windows GOARCH=arm go build -v -o $(BIN)-windows-arm $(SRC)
	GO111MODULE=on GOOS=windows GOARCH=386 go build -v -o $(BIN)-windows-386 $(SRC)
	GO111MODULE=on GOOS=windows GOARCH=amd64 go build -v -o $(BIN)-windows-amd64 $(SRC)

build-darwin:
	GO111MODULE=on GOOS=darwin GOARCH=amd64 go build -v -o $(BIN)-darwin-amd64 $(SRC)

build-all: build-linux build-windows build-darwin

clean:
	rm -f $(BIN) $(BIN)-*
