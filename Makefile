CC = gcc
CPPC = g++
CFLAGS = -Iexten/build/include -Iexten/jubatus-msgpack-rpc/cpp/src
CFLAGS += `pkg-config --cflags webkit2gtk-web-extension-4.0 glib-2.0 gio-2.0`
LFLAGS = -Lexten/build/lib -lpthread
LFLAGS += `pkg-config --libs webkit2gtk-web-extension-4.0 glib-2.0 gio-2.0`
ifdef CLOSURE_COMPILER
PDFJS_METHOD = minified
else
PDFJS_METHOD = generic
endif
OBJ = exten/libgolem.o exten/hints.o exten/rpc.o exten/socket.o
STATICLIBS = exten/build/lib/libmsgpack.a exten/build/lib/libjubatus_mpio.a exten/build/lib/libjubatus_msgpack-rpc.a
MSGPACK = exten/build/lib/libmsgpack.a exten/build/lib/libmsgpackc.a exten/build/include/msgpack exten/build/include/msgpack.h exten/build/include/msgpack.hpp
MPIO = exten/build/lib/libjubatus_mpio.a exten/build/include/jubatus/mp
MSGPACK_RPC = exten/build/lib/libjubatus_msgpack-rpc.a exten/build/include/jubatus/msgpack


.PHONY: all clean pristine

all: data/srv/pdf.js/enabled data/libgolem.so

%.o: %.c $(MSGPACK_RPC)
	$(CC) -c -fPIC -o $@ $< $(CFLAGS)

%.o: %.cpp $(MSGPACK_RPC)
	$(CPPC) -c -fPIC -o $@ $< $(CFLAGS)

$(MSGPACK):
	mkdir -p exten/build
	cd exten/msgpack-c && ./bootstrap
	cd exten/msgpack-c && ./configure --with-pic --prefix=`pwd`/../build
	make -C exten/msgpack-c
	make -C exten/msgpack-c install

$(MPIO):
	mkdir -p exten/build
	cd exten/jubatus-mpio && ./bootstrap
	cd exten/jubatus-mpio && ./configure --with-pic --prefix=`pwd`/../build
	make -C exten/jubatus-mpio
	make -C exten/jubatus-mpio install

$(MSGPACK_RPC): $(MSGPACK) $(MPIO)
	mkdir -p exten/build
	cd exten/jubatus-msgpack-rpc/cpp && ./bootstrap
	cd exten/jubatus-msgpack-rpc/cpp && ./configure --disable-cclog --with-pic --prefix=`pwd`/../../build --with-jubatus-mpio=`pwd`/../../build --with-msgpack=`pwd`/../../build
	make -C exten/jubatus-msgpack-rpc/cpp
	make -C exten/jubatus-msgpack-rpc/cpp install

data/libgolem.so: $(OBJ) $(STATICLIBS)
	$(CPPC) -shared -o $@ $^ $(LFLAGS) -Wl,-z,defs

data/srv/pdf.js/enabled:
	cd pdf.js && node make $(PDFJS_METHOD)
	mkdir -p data/srv
	mv pdf.js/build/$(PDFJS_METHOD)/web -T data/srv/pdf.js/web
	mv pdf.js/build/$(PDFJS_METHOD)/build -T data/srv/pdf.js/build
	touch $@

clean:
	rm exten/*.o
	rm -rf pdf.js/build
	rm -rf exten/build
	git submodule foreach --recursive git clean -dfx

pristine: clean
	git clean -dfx
