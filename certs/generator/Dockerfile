FROM alpine:3.19

RUN apk add --no-cache openssl

COPY generate-certs.sh /generate-certs.sh
RUN chmod +x /generate-certs.sh

ENV CERTS_DIR=/certs

ENTRYPOINT ["/generate-certs.sh"]
