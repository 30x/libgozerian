GOFILES = *.go

all: libgozerian.so 

libgozerian.so: $(GOFILES)
	go build -buildmode=c-shared -o $@ 

test:
	ginkgo --trace

ctest: libgozerian.so
	(cd ./ctests; make test)

alltests: test ctest

race:
	go test -race

cover:
	go test --coverprofile=cov.out
	go tool cover --html=cov.out

graph: states-request.png

states-request.png: states-request.dot
	dot $< -Tpng -o$@

clean:
	go clean
	(cd ./ctests; make clean)
