#include <pthread.h>
#include <errno.h>
#include <fcntl.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/select.h>
#include <sys/time.h>
#include <unistd.h>

extern int local_sdk_speaker_clean_buf_data();
extern int local_sdk_speaker_set_volume(int volume);
extern int local_sdk_speaker_feed_pcm_data(unsigned char *buf, int size);
extern int local_sdk_speaker_set_ap_mode(int mode);
extern int local_sdk_speaker_set_pa_mode(int mode);
extern int *get_speaker_params();
extern int get_speaker_params_run_state();
extern int IMP_AO_QueryChnStat(unsigned int a, unsigned int b, int *buf);
extern void CommandResponse(int fd, const char *res);

static int (*set_pa_mode)(int mode);
static pthread_mutex_t AudioPlayMutex = PTHREAD_MUTEX_INITIALIZER;
static int AudioPlayFd = -1;
static char audioFile[256];
static int Volume = 0;
static int TalkIdleMs = 1500;

enum AudioJobType {
  AUDIO_JOB_NONE = 0,
  AUDIO_JOB_WAV,
  AUDIO_JOB_TALK
};

static enum AudioJobType AudioJob = AUDIO_JOB_NONE;

static long long now_ms(void) {

  struct timeval tv;
  gettimeofday(&tv, NULL);
  return (long long)tv.tv_sec * 1000LL + tv.tv_usec / 1000;
}

int PlayPCM(char *file, int vol) {

  static const int bufLength = 640;
  unsigned char buf[bufLength];
  const unsigned char cmpHeader[] = {
    0x52, 0x49, 0x46, 0x46, 0x00, 0x00, 0x00, 0x00, 0x57, 0x41, 0x56, 0x45, 0x66, 0x6d, 0x74, 0x20,
    0x10, 0x00, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x40, 0x1f, 0x00, 0x00, 0x80, 0x3e, 0x00, 0x00,
    0x02, 0x00, 0x10, 0x00
  };
  const unsigned char cmpData[] = { 0x64, 0x61, 0x74, 0x61 };

  printf("[command] aplay: file:%s\n", file);
  FILE *fp = fopen(file, "rb");
  if(fp == NULL) {
    fprintf(stderr, "[command] aplay err: fopen %s failed!\n", file);
    return -1;
  }
  size_t size = fread(buf, 1, sizeof(cmpHeader), fp);
  if(size != sizeof(cmpHeader)) {
    fprintf(stderr, "[command] aplay err: header size error\n");
    fclose(fp);
    return -1;
  }
  buf[4] = buf[5] = buf[6] = buf[7] = 0;
  if(memcmp(buf, cmpHeader, sizeof(cmpHeader))) {
    fprintf(stderr, "[command] aplay err: header error\n");
    fclose(fp);
    return -1;
  }
  local_sdk_speaker_clean_buf_data();
  local_sdk_speaker_set_volume(vol);
  set_pa_mode(3);
  while(!feof(fp)) {
    size = fread(buf, 1, 8, fp);
    if(size <= 0) break;
    if(size != 8) {
      fprintf(stderr, "[command] aplay err: chunk header size error %zu %lx\n", size, ftell(fp));
      fclose(fp);
      return -1;
    }
    int chunkSize = (buf[7] << 24) | (buf[6] << 16) | (buf[5] << 8) | buf[4];
    if(memcmp(buf, cmpData, sizeof(cmpData))) {
      fseek(fp, chunkSize, SEEK_CUR);
      continue;
    }
    while(!feof(fp) && (chunkSize > 0)) {
      size = fread(buf, 1, bufLength, fp);
      if (size <= 0) break;
      if(size > chunkSize) size = chunkSize;
      chunkSize -= size;
      while(local_sdk_speaker_feed_pcm_data(buf, size)) usleep(100 * 1000);
    }
  }
  fclose(fp);
  int *params = get_speaker_params();
  while(1) {
    int runState = get_speaker_params_run_state();
    if(runState != 3) break;
    int buf[3];
    int stat = IMP_AO_QueryChnStat(params[7], params[8], buf);
    if(stat || !buf[2]) break;
    usleep(100 * 1000);
  }
  usleep(500 * 1000);
  set_pa_mode(0);
  return 0;
}

static int PlayRawPCM(char *file, int vol, int idleMs) {

  static const int bufLength = 640;
  unsigned char buf[bufLength];

  if(idleMs < 200) idleMs = 200;
  if(idleMs > 10000) idleMs = 10000;

  printf("[command] talk: file:%s volume:%d idle:%dms\n", file, vol, idleMs);
  int fd = open(file, O_RDONLY | O_NONBLOCK);
  if(fd < 0) {
    fprintf(stderr, "[command] talk err: open %s failed: %s\n", file, strerror(errno));
    return -1;
  }

  local_sdk_speaker_clean_buf_data();
  local_sdk_speaker_set_volume(vol);
  set_pa_mode(3);

  int res = 0;
  int received = 0;
  long long deadline = now_ms() + idleMs;

  while(1) {
    long long remaining = deadline - now_ms();
    if(remaining <= 0) break;

    fd_set rfds;
    FD_ZERO(&rfds);
    FD_SET(fd, &rfds);
    struct timeval tv;
    tv.tv_sec = remaining / 1000;
    tv.tv_usec = (remaining % 1000) * 1000;

    int ready = select(fd + 1, &rfds, NULL, NULL, &tv);
    if(ready < 0) {
      if(errno == EINTR) continue;
      fprintf(stderr, "[command] talk err: select failed: %s\n", strerror(errno));
      res = -1;
      break;
    }
    if(ready == 0) break;

    ssize_t size = read(fd, buf, sizeof(buf));
    if(size < 0) {
      if(errno == EINTR) continue;
      if(errno == EAGAIN || errno == EWOULDBLOCK) {
        usleep(20 * 1000);
        continue;
      }
      fprintf(stderr, "[command] talk err: read failed: %s\n", strerror(errno));
      res = -1;
      break;
    }
    if(size == 0) {
      if(received) break;
      usleep(20 * 1000);
      continue;
    }

    received = 1;
    deadline = now_ms() + idleMs;
    if(size & 1) size--;
    if(size <= 0) continue;

    int retry = 0;
    while(local_sdk_speaker_feed_pcm_data(buf, size)) {
      if(++retry > 20) {
        fprintf(stderr, "[command] talk err: speaker buffer timeout\n");
        res = -1;
        break;
      }
      usleep(10 * 1000);
    }
    if(res) break;
  }

  close(fd);
  usleep(100 * 1000);
  set_pa_mode(0);
  return res;
}

static void *AudioPlayThread(void *arg) {

  (void)arg;

  while(1) {
    pthread_mutex_lock(&AudioPlayMutex);
    if(AudioPlayFd >= 0) {
      int res;
      if(AudioJob == AUDIO_JOB_TALK) {
        res = PlayRawPCM(audioFile, Volume, TalkIdleMs);
      } else {
        res = PlayPCM(audioFile, Volume);
      }
      CommandResponse(AudioPlayFd, res ? "error" : "ok");
    }
    AudioPlayFd = -1;
    AudioJob = AUDIO_JOB_NONE;
  }
  return NULL;
}

char *AudioPlay(int fd, char *tokenPtr) {

  if(AudioPlayFd >= 0) {
    fprintf(stderr, "[command] aplay err: Previous file is still playing. %d %d\n", AudioPlayFd, fd);
    return "error";
  }

  if(!set_pa_mode) {
    fprintf(stderr, "[command] aplay err: local_sdk_speaker_set_[ap]_mode not found.\n");
    return "error";
  }

  char *p = strtok_r(NULL, " \t\r\n", &tokenPtr);
  if(!p) {
    fprintf(stderr, "[command] aplay err: usage : aplay <wave file> [<volume>]\n");
    return "error";
  }
  strncpy(audioFile, p, 255);
  audioFile[255] = '\0';

  p = strtok_r(NULL, " \t\r\n", &tokenPtr);
  Volume = 40;
  if(p) Volume = atoi(p);

  AudioJob = AUDIO_JOB_WAV;
  AudioPlayFd = fd;
  pthread_mutex_unlock(&AudioPlayMutex);
  return NULL;
}

char *AudioTalk(int fd, char *tokenPtr) {

  if(AudioPlayFd >= 0) {
    fprintf(stderr, "[command] talk err: Previous audio job is still running. %d %d\n", AudioPlayFd, fd);
    return "error";
  }

  if(!set_pa_mode) {
    fprintf(stderr, "[command] talk err: local_sdk_speaker_set_[ap]_mode not found.\n");
    return "error";
  }

  char *p = strtok_r(NULL, " \t\r\n", &tokenPtr);
  if(!p) {
    fprintf(stderr, "[command] talk err: usage : talk <raw pcm file/fifo> [<volume>] [<idle_ms>]\n");
    return "error";
  }
  strncpy(audioFile, p, 255);
  audioFile[255] = '\0';

  p = strtok_r(NULL, " \t\r\n", &tokenPtr);
  Volume = 40;
  if(p) Volume = atoi(p);

  p = strtok_r(NULL, " \t\r\n", &tokenPtr);
  TalkIdleMs = 1500;
  if(p) TalkIdleMs = atoi(p);

  AudioJob = AUDIO_JOB_TALK;
  AudioPlayFd = fd;
  pthread_mutex_unlock(&AudioPlayMutex);
  return NULL;
}

static void __attribute ((constructor)) AudioPlayInit(void) {

  set_pa_mode = local_sdk_speaker_set_ap_mode;
  if(!set_pa_mode) set_pa_mode = local_sdk_speaker_set_pa_mode;
  pthread_mutex_lock(&AudioPlayMutex);
  pthread_t thread;
  if(pthread_create(&thread, NULL, AudioPlayThread, NULL)) {
    fprintf(stderr, "pthread_create error\n");
    pthread_mutex_unlock(&AudioPlayMutex);
    return;
  }
}
