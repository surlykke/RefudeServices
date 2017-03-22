#!/bin/bash
cd `dirname $0`

for dir in RefudeDesktopService RefudeIconService RefudePowerService RefudeWmService; do
	(cd $dir; go install)
done

cp runRefude.sh ../../../../bin 
