FROM alpine:3.18
RUN apk add --no-cache bash curl
COPY run-integration-tests.sh /usr/local/bin/run-integration-tests.sh
RUN chmod +x /usr/local/bin/run-integration-tests.sh
CMD ["/usr/local/bin/run-integration-tests.sh"]
