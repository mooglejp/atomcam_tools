#include <arpa/inet.h>
#include <errno.h>
#include <netinet/in.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/select.h>
#include <sys/socket.h>
#include <sys/time.h>
#include <unistd.h>

#define ATOM_APP_PORT 4000
#define IO_TIMEOUT_SEC 3

static int write_all(int fd, const char *buf, size_t len) {
  while(len > 0) {
    ssize_t n = write(fd, buf, len);
    if(n < 0) {
      if(errno == EINTR) continue;
      return -1;
    }
    buf += n;
    len -= (size_t)n;
  }
  return 0;
}

static int wait_readable(int fd) {
  fd_set rfds;
  struct timeval tv;

  FD_ZERO(&rfds);
  FD_SET(fd, &rfds);
  tv.tv_sec = IO_TIMEOUT_SEC;
  tv.tv_usec = 0;

  return select(fd + 1, &rfds, NULL, NULL, &tv);
}

int main(int argc, char **argv) {
  int fd = socket(AF_INET, SOCK_STREAM, 0);
  if(fd < 0) {
    perror("socket");
    return 1;
  }

  struct sockaddr_in addr;
  memset(&addr, 0, sizeof(addr));
  addr.sin_family = AF_INET;
  addr.sin_port = htons(ATOM_APP_PORT);
  addr.sin_addr.s_addr = htonl(INADDR_LOOPBACK);

  if(connect(fd, (struct sockaddr *)&addr, sizeof(addr)) < 0) {
    perror("connect");
    close(fd);
    return 1;
  }

  for(int i = 1; i < argc; i++) {
    if(i > 1 && write_all(fd, " ", 1) < 0) {
      perror("write");
      close(fd);
      return 1;
    }
    if(write_all(fd, argv[i], strlen(argv[i])) < 0) {
      perror("write");
      close(fd);
      return 1;
    }
  }
  if(write_all(fd, "\n", 1) < 0) {
    perror("write");
    close(fd);
    return 1;
  }
  shutdown(fd, SHUT_WR);

  char buf[1024];
  for(;;) {
    int ready = wait_readable(fd);
    if(ready < 0) {
      if(errno == EINTR) continue;
      perror("select");
      close(fd);
      return 1;
    }
    if(ready == 0) break;

    ssize_t n = read(fd, buf, sizeof(buf));
    if(n < 0) {
      if(errno == EINTR) continue;
      perror("read");
      close(fd);
      return 1;
    }
    if(n == 0) break;
    if(write_all(STDOUT_FILENO, buf, (size_t)n) < 0) {
      perror("stdout");
      close(fd);
      return 1;
    }
  }

  close(fd);
  return 0;
}
