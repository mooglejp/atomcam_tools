#!/bin/sh

HACK_INI=/tmp/hack.ini
PERIODICREC_SDCARD_REMOVE=$(awk -F "=" '/^PERIODICREC_SDCARD_REMOVE *=/ {print $2}' $HACK_INI)
PERIODICREC_SDCARD_REMOVE_DAYS=$(awk -F "=" '/^PERIODICREC_SDCARD_REMOVE_DAYS *=/ {print $2}' $HACK_INI)
ALARMREC_SDCARD_REMOVE=$(awk -F "=" '/^ALARMREC_SDCARD_REMOVE *=/ {print $2}' $HACK_INI)
ALARMREC_SDCARD_REMOVE_DAYS=$(awk -F "=" '/^ALARMREC_SDCARD_REMOVE_DAYS *=/ {print $2}' $HACK_INI)
TIMELAPSE_SDCARD_REMOVE=$(awk -F "=" '/^TIMELAPSE_SDCARD_REMOVE *=/ {print $2}' $HACK_INI)
TIMELAPSE_SDCARD_REMOVE_DAYS=$(awk -F "=" '/^TIMELAPSE_SDCARD_REMOVE_DAYS *=/ {print $2}' $HACK_INI)
PERIODICREC_CIFS_REMOVE=$(awk -F "=" '/^PERIODICREC_CIFS_REMOVE *=/ {print $2}' $HACK_INI)
PERIODICREC_CIFS_REMOVE_DAYS=$(awk -F "=" '/^PERIODICREC_CIFS_REMOVE_DAYS *=/ {print $2}' $HACK_INI)
ALARMREC_CIFS_REMOVE=$(awk -F "=" '/^ALARMREC_CIFS_REMOVE *=/ {print $2}' $HACK_INI)
ALARMREC_CIFS_REMOVE_DAYS=$(awk -F "=" '/^ALARMREC_CIFS_REMOVE_DAYS *=/ {print $2}' $HACK_INI)
TIMELAPSE_CIFS_REMOVE=$(awk -F "=" '/^TIMELAPSE_CIFS_REMOVE *=/ {print $2}' $HACK_INI)
TIMELAPSE_CIFS_REMOVE_DAYS=$(awk -F "=" '/^TIMELAPSE_CIFS_REMOVE_DAYS *=/ {print $2}' $HACK_INI)
ALARMREC_SDCARD=$(awk -F "=" '/^ALARMREC_SDCARD *=/ {print $2}' $HACK_INI)
PERIODICREC_SDCARD=$(awk -F "=" '/^PERIODICREC_SDCARD *=/ {print $2}' $HACK_INI)
HOSTNAME=`hostname`

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

[ "$PERIODICREC_CIFS_REMOVE" != "on" ] && [ "$ALARMREC_CIFS_REMOVE" != "on" ] && [ "$TIMELAPSE_CIFS_REMOVE" != "on" ] && exit 0

/atom_patch/system_bin/mount_cifs.sh || exit -1

if [ "$ALARMREC_CIFS_REMOVE" = "on" ] && [ "$ALARMREC_CIFS_REMOVE_DAYS" != "" ]; then
  remove_old_files /atom/mnt/$HOSTNAME/alarm_record "$ALARMREC_CIFS_REMOVE_DAYS"
  remove_empty_dirs /atom/mnt/$HOSTNAME/alarm_record
fi
if [ "$PERIODICREC_CIFS_REMOVE" = "on" ] && [ "$PERIODICREC_CIFS_REMOVE_DAYS" != "" ]; then
  remove_old_files /atom/mnt/$HOSTNAME/record "$PERIODICREC_CIFS_REMOVE_DAYS"
  remove_empty_dirs /atom/mnt/$HOSTNAME/record
fi
if [ "$TIMELAPSE_CIFS_REMOVE" = "on" ] && [ "$TIMELAPSE_CIFS_REMOVE_DAYS" != "" ]; then
  remove_old_files /atom/mnt/$HOSTNAME/time_lapse "$TIMELAPSE_CIFS_REMOVE_DAYS"
  remove_empty_dirs /atom/mnt/$HOSTNAME/time_lapse
fi
