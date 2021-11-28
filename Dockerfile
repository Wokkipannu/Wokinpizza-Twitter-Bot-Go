FROM golang:alpine

WORKDIR /twitterbot

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY *.go ./

RUN go build -o ./bot

CMD [ "./bot" ]