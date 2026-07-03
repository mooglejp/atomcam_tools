#!/bin/sh

HACK_INI=/tmp/hack.ini
PERIODICREC_SDCARD_REMOVE=$(awk -F "=" '/^PERIODICREC_SDCARD_REMOVE *=/ {print $2}' $HACK_INI)
PERIODICREC_SDCARD_REMOVE_DAYS=$(awk -F "=" '/^PERIODICREC_SDCARD_REMOVE_DAYS *=/ {print $2}' $HACK_INI)
ALARMREC_SDCARD_REMOVE=$(awk -F "=" '/^ALARMREC_SDCARD_REMOVE *=/ {print $2}' $HACK_INI)
ALARMREC_SDCARD_REMOVE_DAYS=$(awk -F "=" '/^ALARMREC_SDCARD_REMOVE_DAYS *=/ {print $2}' $HACK_INI)
TIMELAPSE_SDCARD_REMOVE=$(awk -F "=" '/^TIMELAPSE_SDCARD_REMOVE *=/ {print $2}' $HACK_INI)
TIMELAPSE_SDCARD_REMOVE_DAYS=$(awk -F "=" '/^TIMELAPSE_SDCARD_REMOVE_DAYS *=/ {print $2}' $HACK_INI)
ALARMREC_SDCARD=$(awk -F "=" '/^ALARMREC_SDCARD *=/ {print $2}' $HACK_INI)
PERIODICREC_SDCARD=$(awk -F "=" '/^PERIODICREC_SDCARD *=/ {print $2}' $HACK_INI)

remove_old_files() {
  find "$1" -depth -type f -mtime +"$2" -exec rm -f {} +
}

remove_empty_dirs() {
  find "$1" -depth -type d -mmin +3 -exec rmdir {} + 2>/dev/null
}

if [ "$ALARMREC_SDCARD_REMOVE" = "on" ] && [ "$ALARMREC_SDCARD_REMOVE_DAYS" != "" ]; then
  remove_old_files /media/mmc/alarm_record "$ALARMREC_SDCARD_REMOVE_DAYS"
  remove_empty_dirs /media/mmc/alarm_record
fi
if [ "$PERIODICREC_SDCARD_REMOVE" = "on" ] && [ "$PERIODICREC_SDCARD_REMOVE_DAYS" != "" ]; then
  remove_old_files /media/mmc/record "$PERIODICREC_SDCARD_REMOVE_DAYS"
  remove_empty_dirs /media/mmc/record
fi
if [ "$TIMELAPSE_SDCARD_REMOVE" = "on" ] && [ "$TIMELAPSE_SDCARD_REMOVE_DAYS" != "" ]; then
  remove_old_files /media/mmc/time_lapse "$TIMELAPSE_SDCARD_REMOVE_DAYS"
  remove_empty_dirs /media/mmc/time_lapse
fi
find /media/mmc/time_lapse -depth -type f -name '*._mp4' -mtime +3 -exec rm -f {} +
find /media/mmc/time_lapse -depth -type f -name '*.stsz' -mtime +3 -exec rm -f {} +
