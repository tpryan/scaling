#!/bin/sh
### BEGIN INIT INFO
# Provides:          receiver
# Required-Start:    $local_fs $network $named $time $syslog
# Required-Stop:     $local_fs $network $named $time $syslog
# Default-Start:     2 3 4 5
# Default-Stop:      0 1 6
# Description:       <DESCRIPTION>
### END INIT INFO

SCRIPT=/opt/receiver
RUNAS=root

PIDFILE=/var/run/receiver.pid
LOGFILE=/var/log/receiver/receiver.log
HOSTNAME="$(hostname)"

export REDISHOST=10.187.179.219
export REDISPORT=6379
export SCALE_ENV=GCE 
export PORT=80 
export ENDPOINT=http://34.102.175.184/record

start() {
  if [ -f /var/run/$PIDNAME ] && kill -0 $(cat /var/run/$PIDNAME); then
    echo "Service already running" >&2
    return 1
  fi
  echo "Starting service on ${HOSTNAME}…" >&2
  local CMD="$SCRIPT &> \"$LOGFILE\" & echo \$!"
  su -c "$CMD" $RUNAS > "$PIDFILE"
  echo "Service started" >&2
}

stop() {
  if [ ! -f "$PIDFILE" ] || ! kill -0 $(cat "$PIDFILE"); then
    echo "Service not running on ${HOSTNAME}" >&2
    return 1
  fi
  echo "Stopping service on ${HOSTNAME}…" >&2
  kill -15 $(cat "$PIDFILE") && rm -f "$PIDFILE"
  echo "Service stopped on ${HOSTNAME}" >&2
}

uninstall() {
  echo -n "Are you really sure you want to uninstall this service? That cannot be undone. [yes|No] "
  local SURE
  read SURE
  if [ "$SURE" = "yes" ]; then
    stop
    rm -f "$PIDFILE"
    echo "Notice: log file is not be removed: '$LOGFILE'" >&2
    update-rc.d -f receiver remove
    rm -fv "$0"
  fi
}

case "$1" in
  start)
    start
    ;;
  stop)
    stop
    ;;
  uninstall)
    uninstall
    ;;
  retart)
    stop
    start
    ;;
  *)
    echo "Usage: $0 {start|stop|restart|uninstall}"
esac