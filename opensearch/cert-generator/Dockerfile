FROM alpine:3.18

RUN apk add --no-cache openssl

WORKDIR /certs

COPY generate-certs.sh /usr/local/bin/
RUN chmod +x /usr/local/bin/generate-certs.sh

ENTRYPOINT ["/usr/local/bin/generate-certs.sh"]
