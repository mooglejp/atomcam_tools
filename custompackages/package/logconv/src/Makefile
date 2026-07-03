# logconv for atomcam

CC = ${TARGET_CROSS}gcc
STRIP = $(TARGET_CROSS)strip
TARGETS = logconv atomcmd atomwebcmd atomhookd atomrecpostd

CFLAGS = $(INCLUDES) -std=gnu99 -O2 -Wall -march=mips32r2

all: ${TARGETS}

logconv: logconv.c
	${CC} ${CFLAGS} -o $@ $^
	${STRIP} $@

atomcmd: atomcmd.c
	${CC} ${CFLAGS} -o $@ $^
	${STRIP} $@

atomwebcmd: atomwebcmd.c
	${CC} ${CFLAGS} -o $@ $^
	${STRIP} $@

atomhookd: atomhookd.c
	${CC} ${CFLAGS} -o $@ $^
	${STRIP} $@

atomrecpostd: atomrecpostd.c
	${CC} ${CFLAGS} -o $@ $^
	${STRIP} $@

clean:
	rm -f ${TARGETS} *.o *~

distclean: clean
