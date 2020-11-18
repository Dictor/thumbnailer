FROM golang:1.15-alpine AS go

COPY . /thumbnailer
WORKDIR /thumbnailer
RUN go build

FROM jrottenberg/ffmpeg:4.2-alpine
RUN mkdir -p /thumbnailer
COPY --from=go /thumbnailer /thumbnailer
WORKDIR /thumbnailer
ENTRYPOINT /thumbnailer/thumbnailer -vdir /video
