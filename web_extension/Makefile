CC = gcc
CFLAGS =
CFLAGS += `pkg-config --cflags webkit2gtk-web-extension-4.0 json-glib-1.0`
LFLAGS += `pkg-config --libs json-glib-1.0`

%.o: %.c
	$(CC) -c -fPIC -o $@ $< $(CFLAGS)

libgolem.so: libgolem.o
	$(CC) -shared -o $@ $^ $(LFLAGS)

.PHONY: clean

clean:
	rm *.o
