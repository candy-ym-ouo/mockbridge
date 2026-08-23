# Official multi-architecture Go image with the complete toolchain retained.
FROM golang:1.22

# The repository smoke test uses curl.
RUN apt-get update && apt-get install -y --no-install-recommends curl \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Download dependencies while building so later compilation can use the cache offline.
COPY go.mod go.sum ./
RUN go mod download

# Keep the complete source tree, static admin frontend, migrations, tests, and tools.
COPY . .

# Verify that the checked-in project compiles in the delivery image.
RUN go build ./...

CMD ["bash"]
