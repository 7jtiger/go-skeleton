FROM golang:1.19 as builder

RUN mkdir -p /go/src/go-skeleton

RUN cd /go/src/go-skeleton && git clone -b dev https://jin:xxxxx@github.com/codestates-beb/BEP-go-skeleton-server go-skeleton

RUN cd /go/src/go-skeleton && go mod tidy && go build -o go-skeleton

FROM ubuntu:18.04
WORKDIR /go-skeleton

EXPOSE 16300
RUN ln -sf /usr/share/zoneinfo/Asia/Seoul /etc/localtime
RUN mkdir logs
COPY conf/ conf/
COPY data/ data/
COPY run .
COPY --from=builder /go/src/go-skeleton/go-skeleton /go-skeleton/go-skeleton

CMD ./run s >> logs/go-skeleton-out.log
#ENTRYPOINT ["go-skeleton"]
