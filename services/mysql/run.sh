#!/bin/bash
# Usage: run.sh [ DB user password]
# If args are absent then we'll assume the /provision script will be run
# to create the new DB and user

/etc/init.d/mysql start

if [[ "$1" != "" ]]; then 
  /provision $*
fi

/etc/init.d/mysql stop
mysqld_safe
