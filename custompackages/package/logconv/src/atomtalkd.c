#include <arpa/inet.h>
#include <errno.h>
#include <fcntl.h>
#include <netinet/in.h>
#include <signal.h>
#include <stdarg.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <strings.h>
#include <sys/select.h>
#include <sys/socket.h>
#include <sys/stat.h>
#include <sys/time.h>
#include <sys/types.h>
#include <unistd.h>

#define ATOM_APP_PORT 4000
#define DEFAULT_PORT 4010
#define DEFAULT_VOLUME 40
#define DEFAULT_IDLE_MS 1500
#define DEFAULT_FIFO "/tmp/atomtalk.pcm"

static volatile sig_atomic_t running = 1;

struct session {
  int active;
  int authenticated;
  int fifo_fd;
  int cmd_fd;
  char fifo[256];
  struct sockaddr_in peer;
  long long last_packet_ms;
};

static void on_signal(int sig) {
  (void)sig;
  running = 0;
}

static long long now_ms(void) {
  struct timeval tv;
  gettimeofday(&tv, NULL);
  return (long long)tv.tv_sec * 1000LL + tv.tv_usec / 1000;
}

static void log_msg(const char *fmt, ...) {
  va_list ap;
  va_start(ap, fmt);
  vfprintf(stderr, fmt, ap);
  va_end(ap);
  fputc('\n', stderr);
}

static void copy_str(char *dst, size_t dst_size, const char *src) {
  size_t i = 0;
  if(dst_size == 0) return;
  while(i + 1 < dst_size && src[i]) {
    dst[i] = src[i];
    i++;
  }
  dst[i] = '\0';
}

static void trim(char *s) {
  size_t n = strlen(s);
  while(n > 0 && (s[n - 1] == '\n' || s[n - 1] == '\r' || s[n - 1] == ' ' || s[n - 1] == '\t')) {
    s[--n] = '\0';
  }
}

static int write_all(int fd, const void *buf, size_t len) {
  const char *p = buf;
  while(len > 0) {
    ssize_t n = write(fd, p, len);
    if(n < 0) {
      if(errno == EINTR) continue;
      return -1;
    }
    p += n;
    len -= (size_t)n;
  }
  return 0;
}

static int same_peer(const struct sockaddr_in *a, const struct sockaddr_in *b) {
  return a->sin_addr.s_addr == b->sin_addr.s_addr && a->sin_port == b->sin_port;
}

static void send_reply(int sock, const struct sockaddr_in *dst, const char *msg) {
  sendto(sock, msg, strlen(msg), 0, (const struct sockaddr *)dst, sizeof(*dst));
}

static int connect_atom_command(void) {
  int fd = socket(AF_INET, SOCK_STREAM, 0);
  if(fd < 0) return -1;

  struct sockaddr_in addr;
  memset(&addr, 0, sizeof(addr));
  addr.sin_family = AF_INET;
  addr.sin_port = htons(ATOM_APP_PORT);
  addr.sin_addr.s_addr = htonl(INADDR_LOOPBACK);

  if(connect(fd, (struct sockaddr *)&addr, sizeof(addr)) < 0) {
    close(fd);
    return -1;
  }
  return fd;
}

static int start_talk_command(const char *fifo, int volume, int idle_ms) {
  int fd = connect_atom_command();
  if(fd < 0) return -1;

  char command[512];
  snprintf(command, sizeof(command), "talk %s %d %d\n", fifo, volume, idle_ms);
  if(write_all(fd, command, strlen(command)) < 0) {
    close(fd);
    return -1;
  }
  return fd;
}

static void drain_command_response(int fd, int timeout_ms) {
  char buf[128];
  long long deadline = now_ms() + timeout_ms;
  while(now_ms() < deadline) {
    long long remaining = deadline - now_ms();
    fd_set rfds;
    FD_ZERO(&rfds);
    FD_SET(fd, &rfds);
    struct timeval tv;
    tv.tv_sec = remaining / 1000;
    tv.tv_usec = (remaining % 1000) * 1000;
    int ready = select(fd + 1, &rfds, NULL, NULL, &tv);
    if(ready < 0) {
      if(errno == EINTR) continue;
      return;
    }
    if(ready == 0) return;
    ssize_t n = read(fd, buf, sizeof(buf));
    if(n <= 0) return;
  }
}

static void reset_session(struct session *s) {
  s->active = 0;
  s->authenticated = 0;
  s->fifo_fd = -1;
  s->cmd_fd = -1;
  s->last_packet_ms = 0;
  memset(&s->peer, 0, sizeof(s->peer));
}

static void stop_session(struct session *s) {
  if(s->fifo_fd >= 0) {
    close(s->fifo_fd);
    s->fifo_fd = -1;
  }
  if(s->cmd_fd >= 0) {
    drain_command_response(s->cmd_fd, 2000);
    close(s->cmd_fd);
    s->cmd_fd = -1;
  }
  if(s->fifo[0]) unlink(s->fifo);
  reset_session(s);
}

static int open_fifo_writer(const char *fifo) {
  long long deadline = now_ms() + 2000;
  while(now_ms() < deadline) {
    int fd = open(fifo, O_WRONLY | O_NONBLOCK);
    if(fd >= 0) return fd;
    if(errno != ENXIO && errno != ENOENT && errno != EINTR) return -1;
    usleep(20 * 1000);
  }
  errno = ETIMEDOUT;
  return -1;
}

static int start_session(struct session *s, const struct sockaddr_in *peer, const char *fifo, int volume, int idle_ms) {
  if(s->active) return 0;

  copy_str(s->fifo, sizeof(s->fifo), fifo);
  unlink(s->fifo);
  if(mkfifo(s->fifo, 0600) < 0) {
    log_msg("mkfifo %s failed: %s", s->fifo, strerror(errno));
    return -1;
  }

  s->cmd_fd = start_talk_command(s->fifo, volume, idle_ms);
  if(s->cmd_fd < 0) {
    log_msg("talk command failed: %s", strerror(errno));
    unlink(s->fifo);
    return -1;
  }

  s->fifo_fd = open_fifo_writer(s->fifo);
  if(s->fifo_fd < 0) {
    log_msg("open fifo writer failed: %s", strerror(errno));
    drain_command_response(s->cmd_fd, idle_ms + 1000);
    close(s->cmd_fd);
    s->cmd_fd = -1;
    unlink(s->fifo);
    return -1;
  }

  s->active = 1;
  s->authenticated = 1;
  s->peer = *peer;
  s->last_packet_ms = now_ms();
  log_msg("talk session started from %s:%d", inet_ntoa(peer->sin_addr), ntohs(peer->sin_port));
  return 0;
}

static int parse_control(const unsigned char *buf, ssize_t len, const char *token, int *stop) {
  char text[256];
  if(len < 8 || memcmp(buf, "ATOMTALK", 8) != 0) return 0;
  if(len > 8 && buf[8] != ' ' && buf[8] != '\t' && buf[8] != '\r' && buf[8] != '\n') return 0;

  if((size_t)len >= sizeof(text)) len = sizeof(text) - 1;
  memcpy(text, buf, (size_t)len);
  text[len] = '\0';
  trim(text);

  char *p = text + 8;
  while(*p == ' ' || *p == '\t') p++;
  *stop = 0;

  if(token[0]) {
    char *got = p;
    while(*p && *p != ' ' && *p != '\t') p++;
    if(*p) *p++ = '\0';
    if(strcmp(got, token) != 0) return -1;
    while(*p == ' ' || *p == '\t') p++;
    if(strcasecmp(p, "STOP") == 0) *stop = 1;
    return 1;
  }

  if(strcasecmp(p, "STOP") == 0) {
    *stop = 1;
  } else if(*p && strcasecmp(p, "START") != 0) {
    return -1;
  }
  return 1;
}

static void usage(const char *argv0) {
  fprintf(stderr,
          "Usage: %s [-p port] [-v volume] [-i idle_ms] [-k token] [-f fifo]\n"
          "Receives UDP 8000Hz mono signed 16-bit little-endian PCM and plays it on the camera speaker.\n",
          argv0);
}

int main(int argc, char **argv) {
  int port = DEFAULT_PORT;
  int volume = DEFAULT_VOLUME;
  int idle_ms = DEFAULT_IDLE_MS;
  char token[128] = "";
  char fifo[256] = DEFAULT_FIFO;

  int opt;
  while((opt = getopt(argc, argv, "p:v:i:k:f:h")) != -1) {
    switch(opt) {
      case 'p':
        port = atoi(optarg);
        break;
      case 'v':
        volume = atoi(optarg);
        break;
      case 'i':
        idle_ms = atoi(optarg);
        break;
      case 'k':
        copy_str(token, sizeof(token), optarg);
        break;
      case 'f':
        copy_str(fifo, sizeof(fifo), optarg);
        break;
      case 'h':
      default:
        usage(argv[0]);
        return opt == 'h' ? 0 : 1;
    }
  }

  if(port <= 0 || port > 65535) port = DEFAULT_PORT;
  if(volume < 0) volume = 0;
  if(volume > 100) volume = 100;
  if(idle_ms < 200) idle_ms = 200;
  if(idle_ms > 10000) idle_ms = 10000;

  signal(SIGINT, on_signal);
  signal(SIGTERM, on_signal);
  signal(SIGPIPE, SIG_IGN);

  int sock = socket(AF_INET, SOCK_DGRAM, 0);
  if(sock < 0) {
    perror("socket");
    return 1;
  }

  struct sockaddr_in addr;
  memset(&addr, 0, sizeof(addr));
  addr.sin_family = AF_INET;
  addr.sin_port = htons((unsigned short)port);
  addr.sin_addr.s_addr = htonl(INADDR_ANY);

  if(bind(sock, (struct sockaddr *)&addr, sizeof(addr)) < 0) {
    perror("bind");
    close(sock);
    return 1;
  }

  struct session sess;
  reset_session(&sess);
  log_msg("atomtalkd listening on UDP port %d volume=%d idle=%dms token=%s", port, volume, idle_ms, token[0] ? "on" : "off");

  while(running) {
    long long timeout_ms = 200;
    long long now = now_ms();
    if(sess.active) {
      long long remaining = sess.last_packet_ms + idle_ms - now;
      if(remaining < 0) remaining = 0;
      if(remaining < timeout_ms) timeout_ms = remaining;
    }

    fd_set rfds;
    FD_ZERO(&rfds);
    FD_SET(sock, &rfds);
    struct timeval tv;
    tv.tv_sec = timeout_ms / 1000;
    tv.tv_usec = (timeout_ms % 1000) * 1000;

    int ready = select(sock + 1, &rfds, NULL, NULL, &tv);
    if(ready < 0) {
      if(errno == EINTR) continue;
      break;
    }

    if(ready > 0 && FD_ISSET(sock, &rfds)) {
      unsigned char buf[1500];
      struct sockaddr_in src;
      socklen_t src_len = sizeof(src);
      ssize_t n = recvfrom(sock, buf, sizeof(buf), 0, (struct sockaddr *)&src, &src_len);
      if(n <= 0) continue;

      int stop = 0;
      int control = parse_control(buf, n, token, &stop);
      if(control) {
        if(control < 0) {
          send_reply(sock, &src, "ERR auth\n");
          continue;
        }
        if(sess.active && !same_peer(&sess.peer, &src)) {
          send_reply(sock, &src, "ERR busy\n");
          continue;
        }
        if(stop) {
          if((sess.active || sess.authenticated) && same_peer(&sess.peer, &src)) {
            stop_session(&sess);
          }
          send_reply(sock, &src, "OK stop\n");
          continue;
        }
        sess.authenticated = 1;
        sess.peer = src;
        sess.last_packet_ms = now_ms();
        send_reply(sock, &src, "OK\n");
        continue;
      }

      if(token[0] && (!sess.authenticated || !same_peer(&sess.peer, &src))) {
        continue;
      }
      if(sess.active && !same_peer(&sess.peer, &src)) {
        continue;
      }
      if(!sess.active) {
        if(!sess.authenticated) {
          sess.authenticated = 1;
          sess.peer = src;
        }
        if(start_session(&sess, &src, fifo, volume, idle_ms) < 0) {
          send_reply(sock, &src, "ERR start\n");
          stop_session(&sess);
          continue;
        }
      }

      sess.last_packet_ms = now_ms();
      if(n & 1) n--;
      if(n <= 0) continue;
      ssize_t written = write(sess.fifo_fd, buf, (size_t)n);
      if(written < 0) {
        if(errno == EAGAIN || errno == EWOULDBLOCK) continue;
        log_msg("fifo write failed: %s", strerror(errno));
        stop_session(&sess);
      }
    }

    now = now_ms();
    if(sess.active && now - sess.last_packet_ms >= idle_ms) {
      log_msg("talk session idle timeout");
      stop_session(&sess);
    } else if(!sess.active && sess.authenticated && now - sess.last_packet_ms >= 10000) {
      reset_session(&sess);
    }
  }

  stop_session(&sess);
  close(sock);
  return 0;
}
