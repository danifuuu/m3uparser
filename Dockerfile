# Stage 1: Build
FROM golang:1.23-alpine AS builder

WORKDIR /build

# Cache dependency downloads.
COPY go.mod go.sum ./
RUN go mod download

# Build static binary.
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /m3uparser ./cmd/m3uparser

# Stage 2: Runtime
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /m3uparser /usr/local/bin/m3uparser

# Default data directory (mount volumes here).
ENV DATA_DIR=/data

# M3U source URLs (required, comma-separated).
ENV M3U_URL=

# Processing rules.
ENV SCRUB_HEADER=
ENV SCRUB_DEFAULTS="HD :,SD :"
ENV REMOVE_TERMS=
ENV REMOVE_DEFAULTS="720p,WEB,h264,H264,HDTV,x264,1080p,HEVC,x265,X265"
ENV REPLACE_TERMS=
ENV REPLACE_DEFAULTS="1/2=½,/=-"
ENV EXCLUDE_TERMS=
ENV CLEANERS=
ENV BYPASS_HEADER=

# Output toggles.
ENV LIVE_TV=True
ENV UNSORTED=False
ENV CLEAN_SYNC=False

# Jellyfin integration.
ENV JELLYFIN_URL=
ENV API_KEY=
ENV REFRESH_LIB=False

# Threadfin integration.
ENV TF_USER=
ENV TF_PASS=
ENV TF_HOST=
ENV TF_PORT=

# EPG source URLs (comma-separated).
ENV EPG_URL=

# Logging.
ENV LOG_LEVEL=INFO

ENTRYPOINT ["m3uparser"]
