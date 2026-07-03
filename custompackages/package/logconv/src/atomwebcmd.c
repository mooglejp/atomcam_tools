#include <arpa/inet.h>
#include <errno.h>
#include <fcntl.h>
#include <netinet/in.h>
#include <stdarg.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/select.h>
#include <sys/socket.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <unistd.h>

#define ATOM_APP_PORT 4000
#define WEB_CMD_FIFO "/var/run/webcmd"
#define WEB_RES_FIFO "/var/run/webres"
#define USER_CONFIG "/atom/configs/.user_config"

static void trim(char *s) {
  size_t n = strlen(s);
  while(n > 0 && (s[n - 1] == '\n' || s[n - 1] == '\r' || s[n - 1] == ' ' || s[n - 1] == '\t')) {
    s[--n] = '\0';
  }
}

static int starts_with_key(const char *line, const char *key) {
  size_t n = strlen(key);
  return strncmp(line, key, n) == 0 && (line[n] == '=' || line[n] == ' ' || line[n] == '\t');
}

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
  tv.tv_sec = 3;
  tv.tv_usec = 0;

  return select(fd + 1, &rfds, NULL, NULL, &tv);
}

static int atom_cmd(const char *cmd, char *out, size_t out_size) {
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

  if(write_all(fd, cmd, strlen(cmd)) < 0 || write_all(fd, "\n", 1) < 0) {
    close(fd);
    return -1;
  }
  shutdown(fd, SHUT_WR);

  size_t used = 0;
  while(out_size > 1) {
    int ready = wait_readable(fd);
    if(ready <= 0) break;

    ssize_t n = read(fd, out + used, out_size - 1 - used);
    if(n < 0) {
      if(errno == EINTR) continue;
      close(fd);
      return -1;
    }
    if(n == 0) break;
    used += (size_t)n;
    if(used >= out_size - 1) break;
  }
  out[used] = '\0';
  trim(out);

  close(fd);
  return 0;
}

static void respond(const char *fmt, ...) {
  char buf[1024];
  va_list ap;
  va_start(ap, fmt);
  vsnprintf(buf, sizeof(buf), fmt, ap);
  va_end(ap);

  int fd = open(WEB_RES_FIFO, O_WRONLY);
  if(fd < 0) return;
  write_all(fd, buf, strlen(buf));
  write_all(fd, "\n", 1);
  close(fd);
}

static void drop_caches(void) {
  FILE *fp = fopen("/proc/sys/vm/drop_caches", "w");
  if(fp) {
    fputs("3\n", fp);
    fclose(fp);
  }
}

static int rewrite_flip(const char *mode) {
  FILE *in = fopen(USER_CONFIG, "r");
  FILE *out = fopen(USER_CONFIG "_new", "w");
  if(!in || !out) {
    if(in) fclose(in);
    if(out) fclose(out);
    return -1;
  }

  char line[1024];
  const int normal = strcmp(mode, "normal") == 0;
  while(fgets(line, sizeof(line), in)) {
    if(starts_with_key(line, "verSwitch")) {
      fprintf(out, "verSwitch=%d\n", normal ? 2 : 1);
    } else if(starts_with_key(line, "horSwitch")) {
      fprintf(out, "horSwitch=%d\n", normal ? 2 : 1);
    } else {
      fputs(line, out);
    }
  }

  fclose(in);
  fclose(out);
  if(rename(USER_CONFIG "_new", USER_CONFIG) < 0) return -1;
  drop_caches();
  return 0;
}

static int rewrite_position(const char *pos) {
  double pan = 0.0;
  double tilt = 0.0;
  int h = 0;
  int v = 0;
  if(sscanf(pos, "%lf %lf %d %d", &pan, &tilt, &h, &v) < 4) return -1;

  int x = (int)(pan * 100.0 + 0.5);
  int y = (int)(tilt * 100.0 + 0.5);
  if(h != 0) x = 35000 - x;
  if(v != 0) y = 18000 - y;

  FILE *in = fopen(USER_CONFIG, "r");
  FILE *out = fopen(USER_CONFIG "_new", "w");
  if(!in || !out) {
    if(in) fclose(in);
    if(out) fclose(out);
    return -1;
  }

  char line[1024];
  while(fgets(line, sizeof(line), in)) {
    if(starts_with_key(line, "slide_x")) {
      fprintf(out, "slide_x=%d\n", x);
    } else if(starts_with_key(line, "slide_y")) {
      fprintf(out, "slide_y=%d\n", y);
    } else {
      fputs(line, out);
    }
  }

  fclose(in);
  fclose(out);
  if(rename(USER_CONFIG "_new", USER_CONFIG) < 0) return -1;
  drop_caches();
  return 0;
}

static void read_config_value(const char *key, char *out, size_t out_size) {
  FILE *fp = fopen("/tmp/hack.ini", "r");
  out[0] = '\0';
  if(!fp) return;

  char line[1024];
  size_t key_len = strlen(key);
  while(fgets(line, sizeof(line), fp)) {
    if(strncmp(line, key, key_len) != 0) continue;
    char *p = line + key_len;
    while(*p == ' ' || *p == '\t') p++;
    if(*p != '=') continue;
    p++;
    while(*p == ' ' || *p == '\t') p++;
    snprintf(out, out_size, "%s", p);
    trim(out);
    break;
  }
  fclose(fp);
}

static void shell_quote(const char *src, char *dst, size_t dst_size) {
  size_t used = 0;
  if(dst_size == 0) return;
  dst[used++] = '\'';
  for(; *src && used + 5 < dst_size; src++) {
    if(*src == '\'') {
      memcpy(dst + used, "'\\''", 4);
      used += 4;
    } else {
      dst[used++] = *src;
    }
  }
  if(used < dst_size - 1) dst[used++] = '\'';
  dst[used] = '\0';
}

static void read_first_line(const char *command, char *out, size_t out_size) {
  FILE *fp = popen(command, "r");
  out[0] = '\0';
  if(!fp) return;
  if(fgets(out, out_size, fp)) trim(out);
  pclose(fp);
}

static void run_system(const char *command) {
  int rc = system(command);
  (void)rc;
}

static void start_update_child(void) {
  char custom_zip[32];
  char zip_url[512];
  read_config_value("CUSTOM_ZIP", custom_zip, sizeof(custom_zip));
  read_config_value("CUSTOM_ZIP_URL", zip_url, sizeof(zip_url));

  if(strcmp(custom_zip, "off") == 0 || zip_url[0] == '\0') {
    char latest[512];
    read_first_line("curl -w \"%{redirect_url}\" -s -o /dev/null https://github.com/mnakada/atomcam_tools/releases/latest", latest, sizeof(latest));
    char *tag = strstr(latest, "tag/");
    if(tag) {
      snprintf(zip_url, sizeof(zip_url), "https://github.com/mnakada/atomcam_tools/releases/download/%s/atomcam_tools.zip", tag + 4);
    }
  }

  mkdir("/media/mmc/update", 0777);
  FILE *status = fopen("/tmp/update_status", "w");
  if(status) {
    fputs("0\n", status);
    fclose(status);
  }

  pid_t pid = fork();
  if(pid != 0) return;

  char quoted_url[1200];
  char script[1800];
  shell_quote(zip_url, quoted_url, sizeof(quoted_url));
  snprintf(script, sizeof(script),
           "cd /media/mmc/update && "
           "curl -H 'Cache-Control: no-cache, no-store' -H 'Pragma: no-cache' -L -o atomcam_tools.zip %s 2>&1 | "
           "awk 'BEGIN { RS=\"\\r\"; } /Total/ { next; } { printf(\"%%d\\n\", $3) > \"/tmp/update_status\"; close(\"/tmp/update_status\"); }'; "
           "/scripts/cmd timelapse stop > /dev/null; "
           "sleep 3; killall -SIGUSR2 iCamera_app; sync; sync; sync; reboot",
           quoted_url);
  execl("/bin/sh", "sh", "-c", script, (char *)NULL);
  _exit(127);
}

static void command_with_atom_result(const char *webcmd, const char *atom_prefix, const char *params) {
  char request[512];
  char result[1024];
  if(params && params[0]) {
    snprintf(request, sizeof(request), "%s %s", atom_prefix, params);
  } else {
    snprintf(request, sizeof(request), "%s", atom_prefix);
  }
  if(atom_cmd(request, result, sizeof(result)) < 0) {
    snprintf(result, sizeof(result), "error");
  }
  respond("%s %s %s", webcmd, params ? params : "", result);
}

static void handle_line(char *line) {
  trim(line);
  if(line[0] == '\0') return;

  char *params = strchr(line, ' ');
  if(params) {
    *params++ = '\0';
    while(*params == ' ') params++;
  } else {
    params = "";
  }
  const char *cmd = line;

  if(strcmp(cmd, "reboot") == 0) {
    respond("%s %s OK", cmd, params);
    atom_cmd("timelapse stop", (char[16]){0}, 16);
    run_system("sleep 3; killall -SIGUSR2 iCamera_app; sync; sync; sync; reboot");
    return;
  }
  if(strcmp(cmd, "setCron") == 0) {
    run_system("/scripts/set_crontab.sh");
    respond("%s %s OK", cmd, params);
    return;
  }
  if(strcmp(cmd, "setwebhook") == 0) {
    run_system("killall atomhookd >/dev/null 2>&1; killall atomrecpostd >/dev/null 2>&1; /usr/bin/atomhookd & /usr/bin/atomrecpostd &");
    respond("%s %s OK", cmd, params);
    return;
  }
  if(strcmp(cmd, "hostname") == 0 && params[0]) {
    char host[128];
    snprintf(host, sizeof(host), "%s", params);
    char *dot = strchr(host, '.');
    if(dot) *dot = '\0';
    FILE *fp = fopen("/media/mmc/hostname", "w");
    if(fp) {
      fprintf(fp, "%s\n", host);
      fclose(fp);
    }
    int rc = sethostname(host, strlen(host));
    (void)rc;
    run_system("pidof nmbd >/dev/null 2>&1 && { killall -9 nmbd; nmbd -D; } >/dev/null 2>&1");
    respond("%s %s OK", cmd, params);
    return;
  }
  if(strcmp(cmd, "mp4write") == 0) {
    command_with_atom_result(cmd, "mp4write", params);
    return;
  }
  if(strcmp(cmd, "framerate") == 0) {
    command_with_atom_result(cmd, "video fps", params);
    return;
  }
  if(strcmp(cmd, "bitrate") == 0) {
    command_with_atom_result(cmd, "video bitrate", params);
    return;
  }
  if(strcmp(cmd, "alarm") == 0) {
    command_with_atom_result(cmd, "alarm", params);
    return;
  }
  if(strcmp(cmd, "curl") == 0) {
    command_with_atom_result(cmd, "curl", params);
    return;
  }
  if(strcmp(cmd, "skipRecJpeg") == 0) {
    command_with_atom_result(cmd, "skipRecJpeg", params);
    return;
  }
  if(strcmp(cmd, "flip") == 0) {
    char result[1024];
    char request[512];
    snprintf(request, sizeof(request), "video flip %s", params);
    if(atom_cmd(request, result, sizeof(result)) < 0) snprintf(result, sizeof(result), "error");
    rewrite_flip(params);
    respond("%s %s %s", cmd, params, result);
    return;
  }
  if(strcmp(cmd, "rtspserver") == 0 && params[0]) {
    char command[256];
    snprintf(command, sizeof(command), "/scripts/rtspserver.sh %s", params);
    run_system(command);
    respond("%s %s OK", cmd, params);
    return;
  }
  if(strcmp(cmd, "cruise") == 0) {
    run_system("killall -9 cruise.sh >/dev/null 2>&1; /scripts/cruise.sh &");
    respond("%s %s OK", cmd, params);
    return;
  }
  if(strcmp(cmd, "lighttpd") == 0) {
    respond("%s OK", cmd);
    if(fork() == 0) {
      execl("/bin/sh", "sh", "-c", "sleep 3; /scripts/lighttpd.sh restart", (char *)NULL);
      _exit(127);
    }
    return;
  }
  if(strcmp(cmd, "samba") == 0 && params[0]) {
    char command[256];
    snprintf(command, sizeof(command), "/scripts/samba.sh %s", params);
    run_system(command);
    respond("%s %s OK", cmd, params);
    return;
  }
  if(strcmp(cmd, "sderase") == 0) {
    run_system("busybox rm -rf /media/mmc/record /media/mmc/alarm_record /media/mmc/time_lapse");
    respond("%s %s OK", cmd, params);
    return;
  }
  if(strcmp(cmd, "update_status") == 0) {
    char stat[64] = "-1";
    FILE *fp = fopen("/tmp/update_status", "r");
    if(fp) {
      if(fgets(stat, sizeof(stat), fp)) trim(stat);
      fclose(fp);
    }
    respond("%s %s OK", cmd, stat);
    return;
  }
  if(strcmp(cmd, "update") == 0) {
    start_update_child();
    respond("%s %s OK", cmd, params);
    return;
  }
  if(strcmp(cmd, "posrec") == 0) {
    char pos[1024];
    if(atom_cmd("move", pos, sizeof(pos)) == 0) rewrite_position(pos);
    respond("%s OK", cmd);
    return;
  }
  if(strcmp(cmd, "moveinit") == 0) {
    run_system("/scripts/motor_init");
    respond("%s OK", cmd);
    return;
  }

  respond("%s %s : syntax error", cmd, params);
}

int main(void) {
  unlink(WEB_CMD_FIFO);
  unlink(WEB_RES_FIFO);
  if(mkfifo(WEB_CMD_FIFO, 0666) < 0 && errno != EEXIST) return 1;
  if(mkfifo(WEB_RES_FIFO, 0666) < 0 && errno != EEXIST) return 1;
  chmod(WEB_CMD_FIFO, 0666);
  chmod(WEB_RES_FIFO, 0666);

  for(;;) {
    int fd = open(WEB_CMD_FIFO, O_RDWR);
    if(fd < 0) {
      sleep(1);
      continue;
    }
    FILE *fp = fdopen(fd, "r");
    if(!fp) {
      close(fd);
      sleep(1);
      continue;
    }

    char line[1024];
    while(fgets(line, sizeof(line), fp)) {
      handle_line(line);
    }
    fclose(fp);
  }
}
