FROM golang:1.19 as builder

RUN mkdir -p /go/src/basesk

RUN cd /go/src/basesk && git clone -b dev https://jin:xxxxx@github.com/codestates-beb/BEP-basesk-server basesk

RUN cd /go/src/basesk && go mod tidy && go build -o basesk

FROM ubuntu:18.04
WORKDIR /basesk

EXPOSE 16300
RUN ln -sf /usr/share/zoneinfo/Asia/Seoul /etc/localtime
RUN mkdir logs
COPY conf/ conf/
COPY data/ data/
COPY run .
COPY --from=builder /go/src/basesk/basesk /basesk/basesk

CMD ./run s >> logs/basesk-out.log
#ENTRYPOINT ["basesk"]
