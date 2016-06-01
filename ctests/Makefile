CC = gcc
LD = gcc
CFLAGS = -g -Wall -I.. -I/usr/local/include
LDFLAGS = -g -L.. -L/usr/local/lib 

OBJS = test_main.o c_test.o

all: test

test: ctests
	(export DYLD_LIBRARY_PATH=..; ./ctests)

ctests: $(OBJS)
	$(LD) $(LDFLAGS) -o $@ $(OBJS) -lgozerian -lcunit

clean:
	rm -f ./ctests *.o
