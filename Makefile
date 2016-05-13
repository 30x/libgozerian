GOFILES = gobridge.go commands.go handler.go http.go manager.go request.go

all: gobridge.so

gobridge.so: $(GOFILES)
	go build -buildmode=c-shared -o $@ 

test:
	ginkgo --trace

race:
	go test -race

graph: states.png

states.png: states.dot
	dot -Tpng states.dot -ostates.png

clean:
	go clean
