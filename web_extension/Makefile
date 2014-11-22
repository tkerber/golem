CC = gcc
CFLAGS =
CFLAGS += `pkg-config --cflags webkit2gtk-web-extension-4.0`

%.o: %.c
	$(CC) -c -fPIC -o $@ $< $(CFLAGS)

libgolem.so: web_extension.o
	$(CC) -shared -o $@ $^

.PHONY: clean

clean:
	rm *.o
