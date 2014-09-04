FROM ubuntu:saucy
RUN apt-get -y update && apt-get -y upgrade
RUN apt-get -y install mysql-server
RUN apt-get -y install mysql-client

# Add files needed to setup the DB and its tables
ADD my.cnf /etc/mysql/conf.d/my.cnf
ADD setup.sql /setup.sql
ADD run.sh /run.sh
RUN chmod 755 /run.sh
RUN /run.sh

EXPOSE 3306

# Start DB
ENTRYPOINT mysqld_safe
