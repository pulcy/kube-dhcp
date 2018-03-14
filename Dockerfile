# Build stage
FROM golang:1.10.0-alpine AS build-env
ENV GOPATH /usr/code
ENV CGO_ENABLED 0
WORKDIR /usr/code
COPY vendor/github.com /usr/code/src/github.com/
COPY vendor/golang.org /usr/code/src/golang.org/
COPY vendor/gopkg.in /usr/code/src/gopkg.in/
ADD . /usr/code
RUN go build -installsuffix cgo -o kube-dhcp
RUN ls -al

# final stage
FROM scratch
WORKDIR /app
COPY --from=build-env /usr/code/kube-dhcp /app/
ENTRYPOINT [ "/app/kube-dhcp" ]
