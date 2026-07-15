#!/bin/sh

echo "Cache-Control: no-cache"
echo "Content-Type: text/plain"
echo ""

section() {
  printf 'SECTION=%s\n' "$1"
}

kv() {
  printf 'KV=%s\t%s\n' "$1" "$2"
}

line() {
  printf 'LINE=%s\n' "$*"
}

tail_log() {
  file="$1"
  lines="$2"
  if [ -s "$file" ]; then
    tail -n "$lines" "$file" 2>/dev/null | sed 's/\r//g' | while IFS= read -r l; do
      line "$l"
    done
  else
    line "$file: empty"
  fi
}

section summary
kv "Timestamp" "$(date '+%Y/%m/%d %H:%M:%S')"
kv "Hostname" "$(hostname 2>/dev/null)"
[ -r /etc/atomhack.ver ] && kv "AtomHack" "$(cat /etc/atomhack.ver)"
[ -r /atom/system/bin/app.ver ] && kv "App" "$(awk -F= '/^appver/ {print $2}' /atom/system/bin/app.ver)"
[ -r /atom/configs/.product_config ] && kv "Model" "$(awk -F= '/^PRODUCT_MODEL/ {print $2}' /atom/configs/.product_config)"
kv "Kernel" "$(uname -r)"
[ -r /proc/loadavg ] && kv "Load" "$(cat /proc/loadavg)"
[ -r /proc/uptime ] && kv "UptimeSec" "$(awk '{printf "%d", $1}' /proc/uptime)"
kv "RTSP" "$(pidof v4l2rtspserver >/dev/null && echo running || echo stopped)"
kv "RecordPOST" "$(pidof atomrecpostd >/dev/null && echo running || echo stopped)"
kv "Talk" "$(pidof atomtalkd >/dev/null && echo running || echo stopped)"

section memory
if command -v free >/dev/null 2>&1; then
  free | while IFS= read -r l; do
    line "$l"
  done
else
  awk '/^(MemTotal|MemFree|Buffers|Cached|Shmem|SwapTotal|SwapFree):/ {print}' /proc/meminfo | while IFS= read -r l; do
    line "$l"
  done
fi

section storage
df -k /tmp /media/mmc 2>/dev/null | while IFS= read -r l; do
  line "$l"
done

section network
ifconfig wlan0 2>/dev/null | sed -n '1,4p' | while IFS= read -r l; do
  line "$l"
done
netstat -ltn 2>/dev/null | while IFS= read -r l; do
  case "$l" in
    *:22*|*:80*|*:8554*|*:8080*|*:4010*) line "$l" ;;
  esac
done

section processes
ps | awk '
  NR == 1 ||
  /iCamera_app|v4l2rtspserver|atomrecpostd|atomhookd|atomwebcmd|atomtalkd|lighttpd|sshd|crond|ntpd|wpa_supplicant|sysMonitor/ {
    print
  }
' | while IFS= read -r l; do
  line "$l"
done

section config
awk -F= '
  /^(RTSP_|WEBHOOK_RECORD_|ATOMTALK_|FRAMERATE|BITRATE_|PERIODICREC_SKIP_JPEG)/ {
    print
  }
' /tmp/hack.ini 2>/dev/null | while IFS= read -r l; do
  line "$l"
done

section rtspLog
tail_log /tmp/log/rtspserver.log 40

section recordPostLog
tail_log /tmp/log/record_upload.log 40
