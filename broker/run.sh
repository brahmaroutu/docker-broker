#!/bin/bash
user="docker"
password="docker"

/etc/init.d/mysql start

# Give it time to start
sleep 5

# Set up initial user
mysql <<EOF 
  CREATE USER '$user'@'%' IDENTIFIED by '$password' ;
  GRANT ALL PRIVILEGES ON *.* to '$user'@'%' WITH GRANT OPTION ;
  FLUSH PRIVILEGES ;
EOF

# Set up broker DB and its tables
mysql < /setup.sql

/etc/init.d/mysql stop
