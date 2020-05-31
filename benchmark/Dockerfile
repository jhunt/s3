FROM golang AS build
WORKDIR /src/

COPY . .
ENV CGO_ENABLED=0
RUN go build .

FROM filefrog/hyperfine
RUN dd if=/dev/zero bs=8192 of=10M   count=1280
RUN dd if=/dev/zero bs=8192 of=100M  count=12800
RUN dd if=/dev/zero bs=8192 of=1000M count=128000
COPY --from=build /src/s3 /usr/local/bin/s3
