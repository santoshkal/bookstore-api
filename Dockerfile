FROM wolfi-base AS build

RUN apk update && apk add go

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

ARG COMMIT_SHA
ENV COMMIT_SHA=$COMMIT_SHA
COPY main.go ./
COPY ./pkg ./pkg
COPY ./templates ./templates
COPY ./image ./image
RUN go build -o bookstore -ldflags "-X main.commitSHA=$(COMMIT_SHA)"

RUN echo ${COMMIT_SHA}
EXPOSE 8080

## Deploy 
FROM cgr.dev/chainguard/go:latest

WORKDIR /app

COPY --from=build /home/bookstore ./
COPY --from=build /home/image ./image
COPY --from=build /home/templates/. ./templates

ENTRYPOINT ["./bookstore"]
