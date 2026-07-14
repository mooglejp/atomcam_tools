#!/bin/sh

LOG=/tmp/log/rtspserver.log
HACK_INI=/tmp/hack.ini

log() {
  echo `date +"%Y/%m/%d %H:%M:%S"` ": $*" >> $LOG
}

get_ini() {
  awk -F "=" -v key="$1" '$1 ~ "^[ \t]*" key "[ \t]*$" {print $2}' $HACK_INI | tr -d '\r' | sed 's/^[ \t]*//;s/[ \t]*$//'
}

cmd_log() {
  res=`/scripts/cmd "$@" 2>&1`
  rc=$?
  if [ $rc -ne 0 ]; then
    log "cmd $* failed: rc=$rc res=$res"
    return $rc
  fi
  case "$res" in
    ""|*error*|*timeout*|*connect*|*write*|*read*|*select*)
      log "cmd $* unexpected: rc=$rc res=$res"
    ;;
  esac
  return $rc
}

if [ "$1" = "off" -o "$1" = "restart" ]; then
  cmd_log audio 0 off
  cmd_log audio 1 off
  cmd_log video 0 off
  cmd_log video 1 off
  cmd_log video 2 off
  count=0
  while pidof v4l2rtspserver > /dev/null ; do
    killall v4l2rtspserver > /dev/null 2>&1
    sleep 0.5
    count=$((count + 1))
    if [ $count -ge 20 ]; then
      log "v4l2rtspserver stop timeout"
      break
    fi
  done
  log "v4l2rtspserver stop"
  [ "$1" = "off" ] && exit 0
fi

RTSP_VIDEO0=$(get_ini RTSP_VIDEO0)
RTSP_AUDIO0=$(get_ini RTSP_AUDIO0)
[ "$RTSP_AUDIO0" = "on" ] && RTSP_AUDIO0="S16_BE"
AUDIO0="on"
[ "$RTSP_AUDIO0" = "off" -o "$RTSP_AUDIO0" = "" ] && AUDIO0="off"
RTSP_VIDEO1=$(get_ini RTSP_VIDEO1)
RTSP_AUDIO1=$(get_ini RTSP_AUDIO1)
[ "$RTSP_AUDIO1" = "on" ] && RTSP_AUDIO1="S16_BE"
AUDIO1="on"
[ "$RTSP_AUDIO1" = "off" -o "$RTSP_AUDIO1" = "" ] && AUDIO1="off"
RTSP_VIDEO2=$(get_ini RTSP_VIDEO2)
RTSP_AUDIO2=$(get_ini RTSP_AUDIO2)
[ "$RTSP_AUDIO2" = "on" ] && RTSP_AUDIO2="S16_BE"
AUDIO2="on"
[ "$RTSP_AUDIO2" = "off" -o "$RTSP_AUDIO2" = "" ] && AUDIO2="off"
RTSP_OVER_HTTP=$(get_ini RTSP_OVER_HTTP)
RTSP_AUTH=$(get_ini RTSP_AUTH)
RTSP_USER=$(get_ini RTSP_USER)
RTSP_PASSWD=$(get_ini RTSP_PASSWD)
RTSP_DSCP=$(get_ini RTSP_DSCP)
case "$RTSP_DSCP" in
  ""|*[!0-9]*) RTSP_DSCP=46 ;;
esac
[ "$RTSP_DSCP" -gt 63 ] && RTSP_DSCP=63

if [ "$1" = "watchdog" ]; then
  [ "$RTSP_VIDEO0" = "on" -o "$RTSP_VIDEO1" = "on" -o "$RTSP_VIDEO2" = "on" ] || exit 0
fi

if ! pidof v4l2rtspserver > /dev/null ; then
  [ "$1" != "on" -a "$1" != "restart" -a "$1" != "watchdog" -a "$RTSP_VIDEO0" != "on" -a "$RTSP_VIDEO1" != "on" -a "$RTSP_VIDEO2" != "on" ] && exit 0

  log "RTSP Restart video0=$RTSP_VIDEO0 audio0=$RTSP_AUDIO0 video1=$RTSP_VIDEO1 audio1=$RTSP_AUDIO1 video2=$RTSP_VIDEO2 audio2=$RTSP_AUDIO2 dscp=$RTSP_DSCP"

  cmd_log video 0 $RTSP_VIDEO0
  cmd_log video 1 $RTSP_VIDEO1
  cmd_log video 2 $RTSP_VIDEO2
  [ "$RTSP_VIDEO0" = "on" -a "$AUDIO0" != "off" ] && cmd_log audio 0 on
  [ "$RTSP_VIDEO1" = "on" -a "$AUDIO1" != "off" ] && cmd_log audio 1 on
  [ "$RTSP_VIDEO2" = "on" -a "$AUDIO2" != "off" ] && cmd_log audio 2 on
  if ! pidof v4l2rtspserver > /dev/null ; then
    while netstat -ltn 2> /dev/null | egrep ":(8554|8080)"; do
      sleep 0.5
    done
    log "v4l2rtspserver start"
    [ "$RTSP_OVER_HTTP" = "on" ] && option="-p 8080"
    [ "$RTSP_AUTH" = "on" -a "$RTSP_USER" != "" -a "$RTSP_PASSWD" != "" ] && option="$option -U $RTSP_USER:$RTSP_PASSWD"
    if [ "$RTSP_VIDEO0" = "on" ]; then
      if [ "$AUDIO0" = "off" ]; then
        path="/dev/video0 "
      else
        path="/dev/video0,hw:0,0@$RTSP_AUDIO0 "
      fi
    fi
    if [ "$RTSP_VIDEO1" = "on" ]; then
      if [ "$AUDIO1" = "off" ]; then
        path="$path /dev/video1 "
      else
        path="$path /dev/video1,hw:2,0@$RTSP_AUDIO1 "
      fi
    fi
    if [ "$RTSP_VIDEO2" = "on" ]; then
      if [ "$AUDIO2" = "off" ]; then
        path="$path /dev/video2 "
      else
        path="$path /dev/video2,hw:4,0@$RTSP_AUDIO2 "
      fi
    fi
    if [ "$path" = "" ]; then
      log "v4l2rtspserver start skipped: no video path"
      exit 1
    fi
    sleep 2
    log "exec LIVE555_DSCP=$RTSP_DSCP /usr/bin/v4l2rtspserver $option -C 1 -a S16_LE $path"
    LIVE555_DSCP=$RTSP_DSCP /usr/bin/v4l2rtspserver $option -C 1 -a S16_LE $path >> /tmp/log/rtspserver.log 2>&1 &
  fi
  count=0
  while [ "`pidof v4l2rtspserver`" = "" ]; do
    sleep 0.5
    count=$((count + 1))
    if [ $count -ge 20 ]; then
      log "v4l2rtspserver start timeout"
      exit 1
    fi
  done
  [ "$RTSP_VIDEO0" = "on" ] && cmd_log audio 0 $AUDIO0
  [ "$RTSP_VIDEO1" = "on" ] && cmd_log audio 1 $AUDIO1
  [ "$RTSP_VIDEO2" = "on" ] && cmd_log audio 2 $AUDIO2
fi
exit 0
