#include <errno.h>
#include <fcntl.h>
#include <stdarg.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <time.h>
#include <unistd.h>

#define ATOMAPP_FIFO "/var/run/atomapp"
#define HACK_INI "/tmp/hack.ini"

struct config {
  char webhook_url[512];
  int insecure;
  int atom_log;
  int timelapse_hook;
  int alarm_event;
  int alarm_info;
  int timelapse_event;
};

static void trim(char *s) {
  size_t n = strlen(s);
  while(n > 0 && (s[n - 1] == '\n' || s[n - 1] == '\r' || s[n - 1] == ' ' || s[n - 1] == '\t')) {
    s[--n] = '\0';
  }
}

static int file_exists(const char *path) {
  return access(path, F_OK) == 0;
}

static void run_bg(const char *fmt, ...) {
  char command[1024];
  va_list ap;
  va_start(ap, fmt);
  vsnprintf(command, sizeof(command), fmt, ap);
  va_end(ap);

  pid_t pid = fork();
  if(pid != 0) return;
  execl("/bin/sh", "sh", "-c", command, (char *)NULL);
  _exit(127);
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

static void load_config(struct config *cfg) {
  memset(cfg, 0, sizeof(*cfg));
  cfg->atom_log = file_exists("/media/mmc/atom-log");
  cfg->timelapse_hook = file_exists("/media/mmc/timelapse_hook.sh");

  FILE *fp = fopen(HACK_INI, "r");
  if(!fp) return;

  char line[1024];
  while(fgets(line, sizeof(line), fp)) {
    trim(line);
    char *eq = strchr(line, '=');
    if(!eq) continue;
    *eq++ = '\0';
    while(*eq == ' ' || *eq == '\t') eq++;

    if(strcmp(line, "WEBHOOK_URL") == 0) snprintf(cfg->webhook_url, sizeof(cfg->webhook_url), "%s", eq);
    else if(strcmp(line, "WEBHOOK_INSECURE") == 0) cfg->insecure = strcmp(eq, "on") == 0;
    else if(strcmp(line, "WEBHOOK_ALARM_EVENT") == 0) cfg->alarm_event = strcmp(eq, "on") == 0;
    else if(strcmp(line, "WEBHOOK_ALARM_INFO") == 0) cfg->alarm_info = strcmp(eq, "on") == 0;
    else if(strcmp(line, "WEBHOOK_TIMELAPSE_EVENT") == 0) cfg->timelapse_event = strcmp(eq, "on") == 0;
  }
  fclose(fp);
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

static void post_event(const struct config *cfg, const char *event, const char *data_json) {
  if(cfg->webhook_url[0] == '\0') return;

  char hostname[128] = "atomcam";
  gethostname(hostname, sizeof(hostname) - 1);
  hostname[sizeof(hostname) - 1] = '\0';

  char body[2048];
  if(data_json && data_json[0]) {
    snprintf(body, sizeof(body), "{\"type\":\"%s\", \"device\":\"%s\", \"data\":%s}", event, hostname, data_json);
  } else {
    snprintf(body, sizeof(body), "{\"type\":\"%s\", \"device\":\"%s\"}", event, hostname);
  }

  pid_t pid = fork();
  if(pid != 0) return;

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

static void replace_all(char *s, size_t s_size, const char *from, const char *to) {
  char tmp[1024];
  size_t from_len = strlen(from);
  size_t to_len = strlen(to);
  char *p = s;
  size_t used = 0;

  while(*p && used < sizeof(tmp) - 1) {
    if(strncmp(p, from, from_len) == 0) {
      if(used + to_len >= sizeof(tmp) - 1) break;
      memcpy(tmp + used, to, to_len);
      used += to_len;
      p += from_len;
    } else {
      tmp[used++] = *p++;
    }
  }
  tmp[used] = '\0';
  snprintf(s, s_size, "%s", tmp);
}

static void remove_char(char *s, char c) {
  char *w = s;
  for(char *r = s; *r; r++) {
    if(*r != c) *w++ = *r;
  }
  *w = '\0';
}

static void log_atom_line(const char *line, size_t *log_len, time_t *last_ts, int *paused) {
  FILE *fp;
  time_t now = time(NULL);

  *log_len += strlen(line);
  if(now != *last_ts) {
    if(*last_ts == 0 || *log_len / (size_t)(now - *last_ts) < 1024) {
      *paused = 0;
    } else {
      *paused = 1;
      fp = fopen("/tmp/log/atom.log", "a");
      if(fp) {
        char tbuf[32];
        strftime(tbuf, sizeof(tbuf), "%Y/%m/%d %H:%M:%S", localtime(&now));
        fprintf(fp, "%s : --- Logging is suspended ---\n", tbuf);
        fclose(fp);
      }
    }
    *log_len = 0;
    *last_ts = now;
  }

  if(*paused) return;
  fp = fopen("/tmp/log/atom.log", "a");
  if(fp) {
    fputs(line, fp);
    fputc('\n', fp);
    fclose(fp);
  }
}

static int tokenize(char *line, char **tok, int max_tok) {
  int n = 0;
  char *save = NULL;
  for(char *p = strtok_r(line, " \t", &save); p && n < max_tok; p = strtok_r(NULL, " \t", &save)) {
    tok[n++] = p;
  }
  return n;
}

static void handle_timelapse_hook(const char *line, const struct config *cfg) {
  char copy[1024];
  char *tok[8];
  copy_str(copy, sizeof(copy), line);
  int n = tokenize(copy, tok, 8);
  if(n < 4) return;

  if(strstr(line, "[webhook] time_lapse_event")) {
    if(cfg->timelapse_hook) {
      char count[128];
      snprintf(count, sizeof(count), "%s", tok[3]);
      char *slash = strchr(count, '/');
      if(slash) *slash++ = '\0';
      run_bg("/media/mmc/timelapse_hook.sh %s %s %s %s >/dev/null 2>&1", tok[2], count, slash ? slash : "", n > 4 ? tok[4] : "");
    }
  } else if(strstr(line, "[webhook] time_lapse_finish")) {
    if(n >= 3) run_bg("/scripts/timelapse.sh finish %s", tok[2]);
  }
}

static void handle_line(const char *line, const struct config *cfg, size_t *log_len, time_t *last_ts, int *log_paused) {
  handle_timelapse_hook(line, cfg);

  if(strstr(line, "motor reset done.")) {
    run_bg("/scripts/motor_init reboot");
    if(cfg->atom_log) log_atom_line("motor reset done !!!", log_len, last_ts, log_paused);
    FILE *console = fopen("/dev/console", "w");
    if(console) {
      fputs("motor reset done !!!\n", console);
      fclose(console);
    }
    FILE *done = fopen("/tmp/motor_initialize_done", "w");
    if(done) fclose(done);
  }

  if(cfg->atom_log) log_atom_line(line, log_len, last_ts, log_paused);
  if(cfg->webhook_url[0] == '\0') return;

  if(cfg->alarm_event &&
     (strstr(line, "[aiAlgo] start") ||
      (strstr(line, "alarm_event_handle") && strstr(line, "timestamp")) ||
      (strstr(line, "alarm_event_handle") && strstr(line, "== readly to alarm ==")))) {
    post_event(cfg, "alarmEvent", NULL);
  }

  if(cfg->alarm_info && strstr(line, "[aiAlgo] call_TD_Human_Pet_Predict")) {
    char data[1024];
    const char *p = strstr(line, "Predict ");
    p = p ? p + strlen("Predict ") : line;
    const char *off_end = strstr(p, "] ");
    if(off_end) p = off_end + 2;
    copy_str(data, sizeof(data), p);
    replace_all(data, sizeof(data), "tm:", "");
    replace_all(data, sizeof(data), "|", ",");
    replace_all(data, sizeof(data), "res:", ",");
    remove_char(data, '[');
    remove_char(data, ']');
    char json[1200];
    json_escape_string(data, json, sizeof(json));
    post_event(cfg, "recognitionNotify", json);
  }

  if(cfg->alarm_info && strstr(line, "alarm_event_handle") && strstr(line, "alarmType")) {
    char data[1024];
    const char *p = strstr(line, "alarmType:");
    p = p ? p + strlen("alarmType:") : line;
    copy_str(data, sizeof(data), p);
    char json[1200];
    json_escape_string(data, json, sizeof(json));
    post_event(cfg, "recognitionNotify", json);
  }

  if(cfg->timelapse_event && strstr(line, "[webhook] time_lapse_event")) {
    char data[1024];
    const char *p = strstr(line, "time_lapse_event ");
    p = p ? p + strlen("time_lapse_event ") : line;
    copy_str(data, sizeof(data), p);
    char json[1200];
    json_escape_string(data, json, sizeof(json));
    post_event(cfg, "timelapseEvent", json);
  }
}

int main(void) {
  struct config cfg;
  load_config(&cfg);
  mkdir("/tmp/log", 0777);
  if(mkfifo(ATOMAPP_FIFO, 0666) < 0 && errno != EEXIST) return 1;
  chmod(ATOMAPP_FIFO, 0666);

  size_t log_len = 0;
  time_t last_ts = 0;
  int log_paused = 0;

  for(;;) {
    int fd = open(ATOMAPP_FIFO, O_RDWR);
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

    char line[2048];
    while(fgets(line, sizeof(line), fp)) {
      trim(line);
      handle_line(line, &cfg, &log_len, &last_ts, &log_paused);
    }
    fclose(fp);
  }
}
