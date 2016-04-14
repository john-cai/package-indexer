FROM ubuntu

RUN apt-get update
RUN apt-get -y upgrade
RUN apt-get install -y curl
RUN curl -O https://storage.googleapis.com/golang/go1.6.linux-amd64.tar.gz
RUN tar -xvf go1.6.linux-amd64.tar.gz
RUN mv go /usr/local

ENV GOPATH /go
ENV GOROOT /usr/local/go
ENV PATH /usr/local/go/bin:/go/bin:/usr/local/bin:$PATH

ADD . /go/src/github.com/john-cai/package-indexer
RUN go install github.com/john-cai/package-indexer
ENTRYPOINT /go/bin/package-indexer

EXPOSE 8080
