#include <ctype.h>
#include <dirent.h>
#include <errno.h>
#include <fcntl.h>
#include <stdarg.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/inotify.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <sys/wait.h>
#include <unistd.h>

#define HACK_INI "/tmp/hack.ini"
#define RECORD_DIR "/media/mmc/record"

struct config {
  int record_event;
  int record_upload;
  int insecure;
  int upload_delay_sec;
  int upload_target_sec;
  char webhook_url[512];
  char upload_url[512];
};

struct watch {
  int wd;
  char path[512];
};

struct watches {
  struct watch *items;
  size_t len;
  size_t cap;
};

static void trim(char *s) {
  size_t n = strlen(s);
  while(n > 0 && (s[n - 1] == '\n' || s[n - 1] == '\r' || s[n - 1] == ' ' || s[n - 1] == '\t')) {
    s[--n] = '\0';
  }
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

static int parse_nonnegative_int(const char *s, int fallback, int max) {
  if(!s || !*s) return fallback;
  char *end = NULL;
  long v = strtol(s, &end, 10);
  while(end && (*end == ' ' || *end == '\t')) end++;
  if(end == s || (end && *end) || v < 0) return fallback;
  if(v > max) return max;
  return (int)v;
}

static void log_msg(const char *fmt, ...) {
  FILE *fp = fopen("/tmp/log/record_upload.log", "a");
  if(!fp) return;
  va_list ap;
  va_start(ap, fmt);
  vfprintf(fp, fmt, ap);
  va_end(ap);
  fputc('\n', fp);
  fclose(fp);
}

static void load_config(struct config *cfg) {
  memset(cfg, 0, sizeof(*cfg));
  FILE *fp = fopen(HACK_INI, "r");
  if(!fp) return;

  char line[1024];
  while(fgets(line, sizeof(line), fp)) {
    trim(line);
    char *eq = strchr(line, '=');
    if(!eq) continue;
    *eq++ = '\0';
    while(*eq == ' ' || *eq == '\t') eq++;

    if(strcmp(line, "WEBHOOK_URL") == 0) copy_str(cfg->webhook_url, sizeof(cfg->webhook_url), eq);
    else if(strcmp(line, "WEBHOOK_INSECURE") == 0) cfg->insecure = strcmp(eq, "on") == 0;
    else if(strcmp(line, "WEBHOOK_RECORD_EVENT") == 0) cfg->record_event = strcmp(eq, "on") == 0;
    else if(strcmp(line, "WEBHOOK_RECORD_UPLOAD") == 0) cfg->record_upload = strcmp(eq, "on") == 0;
    else if(strcmp(line, "WEBHOOK_RECORD_UPLOAD_URL") == 0) copy_str(cfg->upload_url, sizeof(cfg->upload_url), eq);
    else if(strcmp(line, "WEBHOOK_RECORD_UPLOAD_DELAY_SEC") == 0) cfg->upload_delay_sec = parse_nonnegative_int(eq, 0, 3600);
    else if(strcmp(line, "WEBHOOK_RECORD_UPLOAD_TARGET_SEC") == 0) cfg->upload_target_sec = parse_nonnegative_int(eq, 0, 3600);
  }
  fclose(fp);

  if(cfg->upload_url[0] == '\0') copy_str(cfg->upload_url, sizeof(cfg->upload_url), cfg->webhook_url);
}

static int has_suffix(const char *s, const char *suffix) {
  size_t sl = strlen(s);
  size_t su = strlen(suffix);
  if(sl < su) return 0;
  return strcmp(s + sl - su, suffix) == 0;
}

static const char *path_for_wd(const struct watches *ws, int wd) {
  for(size_t i = 0; i < ws->len; i++) {
    if(ws->items[i].wd == wd) return ws->items[i].path;
  }
  return NULL;
}

static int add_watch(struct watches *ws, int fd, const char *path) {
  int wd = inotify_add_watch(fd, path, IN_CLOSE_WRITE | IN_MOVED_TO | IN_CREATE | IN_DELETE_SELF | IN_MOVE_SELF);
  if(wd < 0) return -1;

  for(size_t i = 0; i < ws->len; i++) {
    if(ws->items[i].wd == wd) {
      copy_str(ws->items[i].path, sizeof(ws->items[i].path), path);
      return 0;
    }
  }

  if(ws->len == ws->cap) {
    size_t cap = ws->cap ? ws->cap * 2 : 32;
    struct watch *items = realloc(ws->items, cap * sizeof(*items));
    if(!items) return -1;
    ws->items = items;
    ws->cap = cap;
  }
  ws->items[ws->len].wd = wd;
  copy_str(ws->items[ws->len].path, sizeof(ws->items[ws->len].path), path);
  ws->len++;
  return 0;
}

static void add_watch_recursive(struct watches *ws, int fd, const char *path) {
  struct stat st;
  if(stat(path, &st) < 0 || !S_ISDIR(st.st_mode)) return;
  add_watch(ws, fd, path);

  DIR *dir = opendir(path);
  if(!dir) return;
  struct dirent *de;
  while((de = readdir(dir))) {
    if(strcmp(de->d_name, ".") == 0 || strcmp(de->d_name, "..") == 0) continue;
    char child[768];
    snprintf(child, sizeof(child), "%s/%s", path, de->d_name);
    if(stat(child, &st) == 0 && S_ISDIR(st.st_mode)) add_watch_recursive(ws, fd, child);
  }
  closedir(dir);
}

static int build_iso_timestamp(const char *path, char *out, size_t out_size) {
  const char *p = strstr(path, "/record/");
  if(!p) return -1;
  p += strlen("/record/");
  if(strlen(p) < 13) return -1;
  if(!isdigit((unsigned char)p[0]) || !isdigit((unsigned char)p[1]) ||
     !isdigit((unsigned char)p[2]) || !isdigit((unsigned char)p[3]) ||
     !isdigit((unsigned char)p[4]) || !isdigit((unsigned char)p[5]) ||
     !isdigit((unsigned char)p[6]) || !isdigit((unsigned char)p[7]) ||
     p[8] != '/') return -1;
  const char *hour = p + 9;
  const char *slash = strchr(hour, '/');
  if(!slash || slash - hour < 2 || strlen(slash + 1) < 2) return -1;
  snprintf(out, out_size, "%.4s-%.2s-%.2sT%.2s:%.2s:00+09:00", p, p + 4, p + 6, hour, slash + 1);
  return 0;
}

static void sanitize_header_value(const char *src, char *dst, size_t dst_size) {
  size_t i = 0;
  if(dst_size == 0) return;
  while(i + 1 < dst_size && *src) {
    unsigned char c = (unsigned char)*src++;
    dst[i++] = (c == '\r' || c == '\n') ? '_' : (char)c;
  }
  dst[i] = '\0';
}

static void json_escape_string(const char *src, char *dst, size_t dst_size) {
  size_t used = 0;
  if(dst_size == 0) return;
  if(used < dst_size - 1) dst[used++] = '"';
  for(; *src && used + 3 < dst_size; src++) {
    unsigned char c = (unsigned char)*src;
    if(c == '"' || c == '\\') {
      dst[used++] = '\\';
      dst[used++] = (char)c;
    } else if(c >= 0x20) {
      dst[used++] = (char)c;
    }
  }
  if(used < dst_size - 1) dst[used++] = '"';
  dst[used] = '\0';
}

static int wait_child(pid_t pid) {
  int status = 0;
  while(waitpid(pid, &status, 0) < 0) {
    if(errno != EINTR) return -1;
  }
  if(WIFEXITED(status)) return WEXITSTATUS(status);
  return -1;
}

static void post_record_event(const struct config *cfg, const char *path) {
  if(!cfg->record_event || cfg->webhook_url[0] == '\0') return;

  char hostname[128] = "atomcam";
  gethostname(hostname, sizeof(hostname) - 1);
  hostname[sizeof(hostname) - 1] = '\0';

  char json_path[1024];
  json_escape_string(path, json_path, sizeof(json_path));
  char body[1400];
  snprintf(body, sizeof(body), "{\"type\":\"recordEvent\", \"device\":\"%s\", \"data\":%s}", hostname, json_path);

  pid_t pid = fork();
  if(pid != 0) {
    if(pid > 0) wait_child(pid);
    return;
  }

  int devnull = open("/dev/null", O_WRONLY);
  if(devnull >= 0) {
    dup2(devnull, STDOUT_FILENO);
    dup2(devnull, STDERR_FILENO);
    close(devnull);
  }

  if(cfg->insecure) {
    execlp("curl", "curl", "-X", "POST", "-m", "3", "-H", "Content-Type: application/json", "-d", body, "-k", cfg->webhook_url, (char *)NULL);
  } else {
    execlp("curl", "curl", "-X", "POST", "-m", "3", "-H", "Content-Type: application/json", "-d", body, cfg->webhook_url, (char *)NULL);
  }
  _exit(127);
}

static void post_record_file(const struct config *cfg, const char *path) {
  if(!cfg->record_upload || cfg->upload_url[0] == '\0') return;

  struct stat st;
  if(stat(path, &st) < 0 || !S_ISREG(st.st_mode) || st.st_size <= 0) return;

  char iso[40];
  if(build_iso_timestamp(path, iso, sizeof(iso)) < 0) {
    copy_str(iso, sizeof(iso), "1970-01-01T00:00:00+09:00");
  }

  char metadata[96];
  snprintf(metadata, sizeof(metadata), "creation_time=%s", iso);

  char clean_path[512];
  sanitize_header_value(path, clean_path, sizeof(clean_path));
  char header[640];
  snprintf(header, sizeof(header), "x-video-name: %s\r\n", clean_path);

  if(cfg->upload_delay_sec > 0) {
    log_msg("POST delay %ds path=%s", cfg->upload_delay_sec, path);
    sleep((unsigned int)cfg->upload_delay_sec);
  }

  log_msg("POST %s %s target_sec=%d", iso, path, cfg->upload_target_sec);
  pid_t pid = fork();
  if(pid != 0) {
    int rc = pid > 0 ? wait_child(pid) : -1;
    if(rc != 0) log_msg("POST failed rc=%d path=%s", rc, path);
    return;
  }

  char readrate[32];
  char *argv[48];
  int i = 0;
  argv[i++] = "ffmpeg";
  argv[i++] = "-nostdin";
  argv[i++] = "-hide_banner";
  argv[i++] = "-loglevel";
  argv[i++] = "error";
  argv[i++] = "-fflags";
  argv[i++] = "+genpts";
  if(cfg->upload_target_sec > 0) {
    double rate = 60.0 / (double)cfg->upload_target_sec;
    if(rate < 0.001) rate = 0.001;
    if(rate > 100.0) rate = 100.0;
    snprintf(readrate, sizeof(readrate), "%.3f", rate);
    argv[i++] = "-readrate";
    argv[i++] = readrate;
  }
  argv[i++] = "-i";
  argv[i++] = (char *)path;
  argv[i++] = "-threads";
  argv[i++] = "1";
  argv[i++] = "-c:v";
  argv[i++] = "copy";
  argv[i++] = "-fflags";
  argv[i++] = "+genpts";
  argv[i++] = "-avoid_negative_ts";
  argv[i++] = "make_zero";
  argv[i++] = "-c:a";
  argv[i++] = "pcm_mulaw";
  argv[i++] = "-ar";
  argv[i++] = "8000";
  argv[i++] = "-ac";
  argv[i++] = "1";
  argv[i++] = "-metadata";
  argv[i++] = metadata;
  argv[i++] = "-movflags";
  argv[i++] = "frag_keyframe+empty_moov+default_base_moof";
  argv[i++] = "-f";
  argv[i++] = "mov";
  argv[i++] = "-method";
  argv[i++] = "POST";
  argv[i++] = "-content_type";
  argv[i++] = "video/quicktime";
  argv[i++] = "-headers";
  argv[i++] = header;
  argv[i++] = "-rw_timeout";
  argv[i++] = "120000000";
  argv[i++] = (char *)cfg->upload_url;
  argv[i++] = NULL;
  execvp("ffmpeg", argv);
  _exit(127);
}

static void handle_file(const struct config *cfg, const char *path) {
  if(!has_suffix(path, ".mp4")) return;
  post_record_event(cfg, path);
  post_record_file(cfg, path);
}

int main(void) {
  struct config cfg;
  load_config(&cfg);
  if(!cfg.record_event && !cfg.record_upload) return 0;
  if(cfg.record_event && cfg.webhook_url[0] == '\0' && (!cfg.record_upload || cfg.upload_url[0] == '\0')) return 0;
  if(cfg.record_upload && cfg.upload_url[0] == '\0' && !cfg.record_event) return 0;

  mkdir("/tmp/log", 0777);
  int fd = inotify_init();
  if(fd < 0) return 1;
  int flags = fcntl(fd, F_GETFD, 0);
  if(flags >= 0) fcntl(fd, F_SETFD, flags | FD_CLOEXEC);

  struct watches ws;
  memset(&ws, 0, sizeof(ws));
  add_watch_recursive(&ws, fd, RECORD_DIR);

  char buf[8192];
  for(;;) {
    if(ws.len == 0) {
      add_watch_recursive(&ws, fd, RECORD_DIR);
      sleep(2);
      continue;
    }

    ssize_t n = read(fd, buf, sizeof(buf));
    if(n < 0) {
      if(errno == EINTR) continue;
      sleep(1);
      continue;
    }

    for(char *p = buf; p < buf + n;) {
      struct inotify_event *ev = (struct inotify_event *)p;
      const char *base = path_for_wd(&ws, ev->wd);
      if(base && ev->len > 0) {
        char path[768];
        snprintf(path, sizeof(path), "%s/%s", base, ev->name);
        if(ev->mask & IN_ISDIR) {
          if(ev->mask & (IN_CREATE | IN_MOVED_TO)) add_watch_recursive(&ws, fd, path);
        } else if(ev->mask & (IN_CLOSE_WRITE | IN_MOVED_TO)) {
          handle_file(&cfg, path);
        }
      }
      p += sizeof(struct inotify_event) + ev->len;
    }
  }
}
