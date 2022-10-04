FROM golang:1.18
WORKDIR /go/src/github.com/mvazquezc/karma-bot/
ADD cmd /go/src/github.com/mvazquezc/karma-bot/cmd
ADD pkg /go/src/github.com/mvazquezc/karma-bot/pkg
ADD go.mod go.sum /go/src/github.com/mvazquezc/karma-bot/
RUN go mod tidy
RUN GOOS=linux go build -a -installsuffix cgo -o karma-bot cmd/karmabot/main.go

FROM fedora:36
COPY --from=0 /go/src/github.com/mvazquezc/karma-bot/karma-bot /usr/local/bin/karma-bot
CMD ["/usr/local/bin/karma-bot"]
