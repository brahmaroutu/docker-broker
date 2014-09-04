#!/bin/bash
host="localhost"
port=3306
newUser="docker"
password="docker"
newDB="dockerbroker"

/etc/init.d/mysql start

mysqladmin create $newDB
mysql -e "CREATE USER '$newUser'@'%' IDENTIFIED by '$password'"
mysql -e "GRANT ALL PRIVILEGES ON *.* to '$newUser'@'%' WITH GRANT OPTION"
mysql -e "FLUSH PRIVILEGES"
mysql -u$newUser -p$password < ~/go/src/github.com/brahmaroutu/docker-broker/broker/setup.sql 

/etc/init.d/mysql stop

mysqld_safe &
cd /go/src/github.com/brahmaroutu/docker-broker/broker

sleep 10
perl -pi.bak -e 's/^#(\s+)StrictHostKeyChecking ask/$1 StrictHostKeyChecking no/g' /etc/ssh/ssh_config
go run main.go
