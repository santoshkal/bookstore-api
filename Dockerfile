FROM golang:1.25-alpine AS build

WORKDIR /home

COPY go.mod go.sum ./

RUN go mod download

ARG COMMIT_SHA
ENV COMMIT_SHA=$COMMIT_SHA
COPY main.go ./
COPY pkg ./pkg
COPY templates ./templates
COPY image ./image
RUN go build -o bookstore -ldflags "-X main.commitSHA=$(COMMIT_SHA)"

RUN echo ${COMMIT_SHA}
EXPOSE 8080

## Deploy 
FROM alpine:latest 

WORKDIR /root

COPY --from=build /home/bookstore /root
COPY --from=build /home/image /root/image
COPY --from=build /home/templates/. /root/templates

ENTRYPOINT ["./bookstore"]
