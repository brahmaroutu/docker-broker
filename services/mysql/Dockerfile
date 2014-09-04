FROM ubuntu:saucy
RUN apt-get -y update && apt-get -y upgrade
RUN apt-get -y install mysql-server
RUN apt-get -y install mysql-client

ADD my.cnf /etc/mysql/conf.d/my.cnf
ADD run.sh /run.sh
ADD provision /provision
ADD bind /bind
ADD unbind /unbind
ADD deprovision /deprovision

RUN chmod 755 /run.sh
RUN chmod 755 /provision
RUN chmod 755 /bind
RUN chmod 755 /unbind
RUN chmod 755 /deprovision

EXPOSE 3306

# Use CMD instead of ENTRYPOINT so we can override it easily to either
# debug (via bash) or to pass in default DB/user/password args
CMD /run.sh
