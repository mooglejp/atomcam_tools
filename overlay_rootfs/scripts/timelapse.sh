#!/bin/sh

HACK_INI=/tmp/hack.ini
HOSTNAME=`hostname`
WEBHOOK_URL=$(awk -F "=" '/^WEBHOOK_URL *=/ {print $2}' $HACK_INI)
[ "$(awk -F "=" '/^WEBHOOK_INSECURE *=/ {print $2}' $HACK_INI)" = "on" ] && INSECURE_FLAG="-k "

if [ "$1" = "finish" ] ; then
  TIMELAPSE_SDCARD=$(awk -F "=" '/^TIMELAPSE_SDCARD *=/ {print $2}' $HACK_INI)
  WEBHOOK_TIMELAPSE_FINISH=$(awk -F "=" '/^WEBHOOK_TIMELAPSE_FINISH *=/ {print $2}' $HACK_INI)
  (
    if [ "$TIMELAPSE_SDCARD" = "on" ]; then
      STORAGE="${STORAGE}, \"sdcardFile\":\"${2##*media/mmc/}\""
    else
      rm $2
      find /media/mmc/time_lapse -depth -type d -exec rmdir {} + 2>/dev/null
    fi
    if [ "$WEBHOOK_URL" != "" ] && [ "$WEBHOOK_TIMELAPSE_FINISH" = "on" ]; then
      /usr/bin/curl -X POST -m 3 -H "Content-Type: application/json" -d "{\"type\":\"timelapseFinish\", \"device\":\"${HOSTNAME}\"${STORAGE}}" $INSECURE_FLAG $WEBHOOK_URL > /dev/null 2>&1
    fi
  ) &
  exit 0
fi

if [ "$1" = "start" ] ; then
  TIMELAPSE_SDCARD=$(awk -F "=" '/^TIMELAPSE_SDCARD *=/ {print $2}' $HACK_INI)
  [ "$TIMELAPSE_SDCARD" = "on" ] || exit 0

  WEBHOOK_TIMELAPSE_START=$(awk -F "=" '/^WEBHOOK_TIMELAPSE_START *=/ {print $2}' $HACK_INI)
  TIMELAPSE_FPS=$(awk -F "=" '/^TIMELAPSE_FPS *=/ {print $2}' $HACK_INI)
  TIMELAPSE_SDCARD_PATH=$(awk -F "=" '/^TIMELAPSE_SDCARD_PATH *=/ {print $2}' $HACK_INI)
  TIMELAPSE_FILE=`date +"/media/mmc/time_lapse/$TIMELAPSE_SDCARD_PATH.mp4"`
  TIMELAPSE_DIR=${TIMELAPSE_FILE%/*}
  mkdir -p $TIMELAPSE_DIR

  [ -f /media/mmc/timelapse_hook.sh ] && /media/mmc/timelapse_hook.sh $TIMELAPSE_FILE start $3 $4

  res=`/scripts/cmd timelapse $TIMELAPSE_FILE $2 $3 $TIMELAPSE_FPS $4`
  [ "$res" = "ok" ] || exit 1
  if [ "$WEBHOOK_URL" != "" ] && [ "$WEBHOOK_TIMELAPSE_START" = "on" ]; then
    /usr/bin/curl -X POST -m 3 -H "Content-Type: application/json" -d "{\"type\":\"timelapseStart\", \"device\":\"${HOSTNAME}\"}" $INSECURE_FLAG $WEBHOOK_URL > /dev/null 2>&1
  fi
fi
