#!/usr/bin/env bash
# Very simple script to run refude services
# Call from inside a DE-session, as (at least) PATH and DISPLAY variables
# are needed by the services.
#

GOPATH="${GOPATH:-$HOME/go}"
HAPROXYCFG="$GOPATH/src/github.com/surlykke/RefudeServices/haproxy.cfg"

# Run refudeservices.
for app in RefudeDesktopService RefudeIconService RefudePowerService RefudeWmService RefudeConnmanService ; do
	nohup $app >/dev/null 2>/dev/null &
done

# Start haproxy
HAPIDFILE=$XDG_RUNTIME_DIR/refude-haproxy.pid
ps --pid `cat $HAPIDFILE` >/dev/null || haproxy -f $HAPROXYCFG -p $HAPIDFILE
