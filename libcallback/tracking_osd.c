#include <pthread.h>
#include <stdio.h>
#include <string.h>
#include <unistd.h>

#define TRACKING_OSD_GROUP 0
#define TRACKING_OSD_REGION_COUNT 3

struct RgnInfoSt {
  int type;
  int x1;
  int y1;
  int x2;
  int y2;
  int pixfmt;
  int color;
  int width;
};

struct RgnGrpInfoSt {
  int show;
  int x;
  int y;
  float scalex;
  float scaley;
  int galphaEn;
  int fgAlpha;
  int bgAlpha;
  int layer;
};

extern int swing;
extern int GetUserConfig(const char *key);
extern int IMP_OSD_ShowRgn(int handle, int grp, int display);
extern int IMP_OSD_CreateRgn(struct RgnInfoSt *info);
extern int IMP_OSD_RegisterRgn(int handle, int grp, struct RgnGrpInfoSt *grpInfo);
extern int IMP_OSD_UnRegisterRgn(int handle, int grp);
extern void IMP_OSD_DestroyRgn(int handle);

static pthread_mutex_t TrackingOSDMutex = PTHREAD_MUTEX_INITIALIZER;
static int TrackingOSDHandles[TRACKING_OSD_REGION_COUNT] = {
  [0 ... TRACKING_OSD_REGION_COUNT - 1] = -1
};
static int TrackingOSDHandleCount = 0;
static int TrackingOSDAppliedState = -1;

static void DestroyTrackingOSD(void) {

  for(int i = 0; i < TrackingOSDHandleCount; i++) {
    if(TrackingOSDHandles[i] < 0) continue;
    IMP_OSD_ShowRgn(TrackingOSDHandles[i], TRACKING_OSD_GROUP, 0);
    IMP_OSD_UnRegisterRgn(TrackingOSDHandles[i], TRACKING_OSD_GROUP);
    IMP_OSD_DestroyRgn(TrackingOSDHandles[i]);
    TrackingOSDHandles[i] = -1;
  }
  TrackingOSDHandleCount = 0;
}

static int CreateTrackingOSD(void) {

  /*
   * Draw a small target mark in the top-right corner of the 640x360 OSD
   * coordinate space. The group scales it to the 1920x1080 main stream in
   * the same way as the existing center mark.
   */
  static const int Shapes[TRACKING_OSD_REGION_COUNT][5] = {
    { 2, 588,  13, 618,  43 }, // rectangle
    { 1, 584,  28, 622,  28 }, // horizontal crosshair
    { 1, 603,   9, 603,  47 }, // vertical crosshair
  };

  struct RgnGrpInfoSt grpInfo;
  memset(&grpInfo, 0, sizeof(grpInfo));
  grpInfo.show = 0;
  grpInfo.scalex = 3.0;
  grpInfo.scaley = 3.0;
  grpInfo.layer = 3;

  struct RgnInfoSt info;
  memset(&info, 0, sizeof(info));
  info.pixfmt = 8;
  info.color = 0xffffa060;    // same visible red used by the center mark
  info.width = 2;

  for(int i = 0; i < TRACKING_OSD_REGION_COUNT; i++) {
    info.type = Shapes[i][0];
    info.x1 = Shapes[i][1];
    info.y1 = Shapes[i][2];
    info.x2 = Shapes[i][3];
    info.y2 = Shapes[i][4];

    int handle = IMP_OSD_CreateRgn(&info);
    if(handle < 0) {
      DestroyTrackingOSD();
      return -1;
    }
    TrackingOSDHandles[TrackingOSDHandleCount++] = handle;
    if(IMP_OSD_RegisterRgn(handle, TRACKING_OSD_GROUP, &grpInfo)) {
      DestroyTrackingOSD();
      return -2;
    }
  }
  return 0;
}

int TrackingOSDSet(int enabled) {

  enabled = !!enabled;
  pthread_mutex_lock(&TrackingOSDMutex);

  if(TrackingOSDAppliedState == enabled) {
    pthread_mutex_unlock(&TrackingOSDMutex);
    return 0;
  }

  if(enabled && !TrackingOSDHandleCount) {
    int ret = CreateTrackingOSD();
    if(ret) {
      pthread_mutex_unlock(&TrackingOSDMutex);
      return ret;
    }
  }

  int ret = 0;
  for(int i = 0; i < TrackingOSDHandleCount; i++) {
    if(IMP_OSD_ShowRgn(TrackingOSDHandles[i], TRACKING_OSD_GROUP, enabled)) ret = -3;
  }
  if(ret) {
    DestroyTrackingOSD();
    TrackingOSDAppliedState = -1;
  } else {
    TrackingOSDAppliedState = enabled;
  }

  pthread_mutex_unlock(&TrackingOSDMutex);
  return ret;
}

static void *TrackingOSDThread(void *arg) {

  (void)arg;
  unsigned long failureStreak = 0;

  /* Wait for iCamera_app to initialize its user configuration and OSD group. */
  sleep(2);
  while(1) {
    if(swing) {
      int state = GetUserConfig("TrackSwitch");
      if((state == 1) || (state == 2)) {
        int ret = TrackingOSDSet(state == 1);
        if(ret) {
          failureStreak++;
          if((failureStreak <= 3) || (failureStreak == 10) || !(failureStreak % 60)) {
            fprintf(stderr, "tracking osd: update failed: %d streak=%lu\n", ret, failureStreak);
          }
        } else if(failureStreak) {
          fprintf(stderr, "tracking osd: recovered after %lu failures\n", failureStreak);
          failureStreak = 0;
        }
      }
    }
    sleep(1);
  }
  return NULL;
}

static void __attribute ((constructor)) tracking_osd_init(void) {

  pthread_t thread;
  if(pthread_create(&thread, NULL, TrackingOSDThread, NULL)) {
    fprintf(stderr, "tracking osd: pthread_create error\n");
  }
}
