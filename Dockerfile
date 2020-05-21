FROM golang:latest
WORKDIR /go/src/github.com/mvazquezc/karma-bot/
ADD cmd /go/src/github.com/mvazquezc/karma-bot/cmd
ADD pkg /go/src/github.com/mvazquezc/karma-bot/pkg
RUN go get github.com/slack-go/slack && go get github.com/mattn/go-sqlite3
RUN GOOS=linux go build -a -installsuffix cgo -o karma-bot cmd/karmabot/main.go

FROM centos
COPY --from=0 /go/src/github.com/mvazquezc/karma-bot/karma-bot /usr/local/bin/karma-bot
CMD ["/usr/local/bin/karma-bot"]
