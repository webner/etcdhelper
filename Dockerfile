FROM golang:1.11-alpine

WORKDIR /go/src/app
COPY . .
RUN go build .


FROM alpine:latest  
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=0 /go/src/app/app etcdhelper
ENTRYPOINT ["/root/etcdhelper"]  
