FROM golang:1.21-alpine3.18

# Install required packages for testing
RUN apk add --no-cache bash curl git coreutils findutils grep

# Create work directory
WORKDIR /work

# Copy test script
COPY run-integration-tests.sh /usr/local/bin/run-integration-tests.sh
RUN chmod +x /usr/local/bin/run-integration-tests.sh

CMD ["/usr/local/bin/run-integration-tests.sh"]
