GOFILES = gobridge.go commands.go handler.go http.go manager.go request.go

all: gobridge.so

gobridge.so: $(GOFILES)
	go build -buildmode=c-shared -o $@ 

test:
	ginkgo --trace

race:
	go test -race

clean:
	go clean
