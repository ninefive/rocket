FROM alpine:latest

RUN apk update && apk upgrade && \
    apk add --no-cache bash git openssh

WORKDIR /rocket

ADD dist/rocket /bin/rocket

CMD ["rocket"]
