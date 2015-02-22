CC = gcc
CFLAGS = -Iexten/jsonrpC/build/jsonrpc-0.1/include
CFLAGS += `pkg-config --cflags webkit2gtk-web-extension-4.0 glib-2.0 gio-2.0 yajl`
LFLAGS = -Lexten/jsonrpC/build/jsonrpc-0.1/lib
LFLAGS += `pkg-config --libs webkit2gtk-web-extension-4.0 glib-2.0 gio-2.0 yajl`
ifdef CLOSURE_COMPILER
PDFJS_METHOD = minified
else
PDFJS_METHOD = generic
endif
OBJ = exten/libgolem.o\
	exten/hints.o\
	exten/rpc.o\
	exten/jsonrpc_plugin_g_io_channel.o\
	exten/jsonrpc_plugin_yajl.o

.PHONY: all clean

all: data/srv/pdf.js/enabled data/libgolem.so

%.o: %.c exten/jsonrpC/build
	$(CC) -c -fPIC -o $@ $< $(CFLAGS)

exten/jsonrpC/build:
	mkdir -p $@
	# We have to explicitly set the websockets library to be empty. It will still
	# complain, but it will build it.
	cd $@; cmake -DWEBSOCKETS_LIBRARY= ..
	make -C $@ jsonrpc_s

data/libgolem.so: $(OBJ)
	$(CC) -shared -o $@ $^ $(LFLAGS)

data/srv/pdf.js/enabled: pdf.js/
	cd $< && node make $(PDFJS_METHOD)
	mkdir -p data/srv
	mv pdf.js/build/$(PDFJS_METHOD)/web -T data/srv/pdf.js/web
	mv pdf.js/build/$(PDFJS_METHOD)/build -T data/srv/pdf.js/build
	touch $@

pdf.js/:
	git clone --depth 1 git://github.com/mozilla/pdf.js.git $@

clean:
	rm exten/*.o
	rm -rf exten/jsonrpC/build
	rm -rf pdf.js
