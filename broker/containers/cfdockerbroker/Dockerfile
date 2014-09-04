FROM ubuntu:saucy
RUN apt-get -y update && apt-get -y upgrade
RUN apt-get -y install mysql-server
RUN apt-get -y install mysql-client
RUN /etc/init.d/mysql start

ADD my.cnf /etc/mysql/conf.d/my.cnf
ADD run.sh /run.sh
ADD broker /broker

RUN chmod 755 /run.sh

RUN apt-get -y install sshpass
RUN apt-get -y install git
RUN apt-get -y install wget
RUN apt-get -y update && apt-get -y upgrade
RUN apt-get -y install gcc
RUN wget --no-check-certificate https://storage.googleapis.com/golang/go1.3.1.linux-amd64.tar.gz
RUN tar -C /usr/local -xzf go1.3.1.linux-amd64.tar.gz 
ENV  GOROOT /usr/local/go
RUN mkdir ~/go
ENV PATH $PATH:$GOROOT/bin
ENV  GOPATH $HOME/go
RUN export PATH=$PATH:$GOPATH/bin
RUN go get "github.com/go-sql-driver/mysql"
RUN go get  "github.com/gorilla/context"
RUN go get "github.com/gorilla/mux"
RUN go get "github.com/lib/pq"
RUN go get "github.com/mattn/go-sqlite3"
  
RUN mkdir -p ~/go/src/github.com/brahmaroutu/docker-broker/broker
RUN cp -R /broker/* ~/go/src/github.com/brahmaroutu/docker-broker/broker
RUN cd ~/go/src/github.com/brahmaroutu/docker-broker/broker

EXPOSE 3306
EXPOSE 9998

#ENTRYPOINT /run.sh
CMD /run.sh 
