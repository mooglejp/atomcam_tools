#!/bin/sh

if ! pidof ntpd > /dev/null ; then
  echo $(date +"%Y/%m/%d %H:%M:%S : ntpd not running -> restart")
  /etc/init.d/S42ntpd start
fi

# save current time for the boot-time fallback (no RTC battery)
if grep -q ' /media/mmc ' /proc/mounts ; then
  echo "utc=$(date +%s)" > /media/mmc/time.ini
fi
