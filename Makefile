CC = gcc
CFLAGS = -Iexten/jsonrpC/build/jsonrpc-0.1/include
CFLAGS += `pkg-config --cflags webkit2gtk-web-extension-4.0 glib-2.0 gio-2.0`
LFLAGS = -Lexten/jsonrpC/build/jsonrpc-0.1/lib
LFLAGS += `pkg-config --libs webkit2gtk-web-extension-4.0 glib-2.0 gio-2.0`
# Path to google's closure compiler for javascript.
CLOSURE_COMPILER=/usr/share/java/closure-compiler/closure-compiler.jar

.PHONY: all clean

all: data/srv/pdf.js/enabled data/libgolem.so

%.o: %.c exten/jsonrpC/build
	$(CC) -c -fPIC -o $@ $< $(CFLAGS)

exten/jsonrpC/build:
	mkdir -p $@
	cd $@; cmake ..
	make -C $@ jsonrpc_s

data/libgolem.so: exten/libgolem.o exten/hints.o exten/rpc.o
	$(CC) -shared -o $@ $^ $(LFLAGS)

data/srv/pdf.js/enabled: pdf.js/
	cd $< && CLOSURE_COMPILER=$(CLOSURE_COMPILER) node make minified
	mkdir -p data/srv
	mv pdf.js/build/minified/web -T data/srv/pdf.js/web
	mv pdf.js/build/minified/build -T data/srv/pdf.js/build
	touch $@

pdf.js/:
	git clone --depth 1 git://github.com/mozilla/pdf.js.git $@

clean:
	rm exten/*.o
	rm -rf exten/jsonrpC/build
	rm -rf pdf.js
