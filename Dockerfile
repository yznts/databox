# -------------
# build stage
# -------------
FROM golang:alpine AS build

# System deps
RUN apk add build-base

# Attach sources
WORKDIR /src
ADD . /src

# Build
RUN go build -o bin/databox ./cmd/databox

# -------------
# runtime stage
# -------------
FROM alpine

# Copy utilities
COPY --from=build /src/bin/* /usr/local/bin/

# Set workdir
WORKDIR /root

# Entrypoint
ENTRYPOINT ["sh"]