FROM alpine:3.18
RUN apk add --no-cache bash curl
COPY generate-test-audio.sh /usr/local/bin/generate-test-audio.sh
RUN chmod +x /usr/local/bin/generate-test-audio.sh
CMD ["/usr/local/bin/generate-test-audio.sh", "/output"]
