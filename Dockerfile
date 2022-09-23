# syntax=docker/dockerfile:1

# First stage. Building a binary
# -----------------------------------------------------------------------------

# Base image for builder is debian 11 with golang 1.18+ pre-installed
FROM golang:1.18.1-bullseye AS builder

# Download the source code
RUN git clone https://github.com/Threadfin/Threadfin.git /src
WORKDIR /src
RUN git checkout beta

# Install dependencies
RUN go mod tidy && go mod vendor

# Compile
RUN go build threadfin.go

# Second stage. Creating an image
# -----------------------------------------------------------------------------

# Base image is a latest stable debian
FROM debian

ARG BUILD_DATE
ARG VCS_REF
ARG THREADFIN_PORT=34400
ARG THREADFIN_VERSION

LABEL org.label-schema.build-date="{$BUILD_DATE}" \
      org.label-schema.name="Threadfin" \
      org.label-schema.description="Dockerized Threadfin" \
      org.label-schema.url="https://hub.docker.com/r/fyb3roptik/threadfin/" \
      org.label-schema.vcs-ref="{$VCS_REF}" \
      org.label-schema.vcs-url="https://github.com/Threadfin/Threadfin" \
      org.label-schema.vendor="Threadfin" \
      org.label-schema.version="{$XTEVE_VERSION}" \
      org.label-schema.schema-version="1.0"

ENV THREADFIN_BIN=/home/threadfin/bin
ENV THREADFIN_CONF=/home/threadfin/conf
ENV THREADFIN_HOME=/home/threadfin
ENV THREADFIN_TEMP=/tmp/threadfin
ENV THREADFIN_UID=31337
ENV THREADFIN_USER=threadfin

# Create the user to run inside the container
RUN adduser --uid $THREADFIN_UID $THREADFIN_USER

# Add binary to PATH
ENV PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:$THREADFIN_BIN

# Set working directory
WORKDIR $THREADFIN_HOME

# Update package lists
RUN apt-get update

# Install CA certificates
RUN apt-get install --yes ca-certificates

# Add VLC and FFMPEG support
RUN apt-get install --yes vlc-bin ffmpeg

# Copy built binary from builder image
COPY --from=builder [ "/src/threadfin", "${THREADFIN_BIN}/" ]

# Set binary permissions
RUN chmod +rx $THREADFIN_BIN/threadfin

# Create XML cache directory
RUN mkdir $THREADFIN_HOME/cache

# Create working directories for xTeVe
RUN mkdir $THREADFIN_CONF
RUN chmod a+rwX $THREADFIN_CONF
RUN mkdir $THREADFIN_TEMP
RUN chmod a+rwX $THREADFIN_TEMP

# Configure container volume mappings
VOLUME $THREADFIN_CONF
VOLUME $THREADFIN_TEMP

# Ensure the container user has ownership of home dir
RUN chown -R $THREADFIN_USER $THREADFIN_HOME

# Switch users to the xTeVe container user
USER $THREADFIN_USER

# Run the xTeVe executable
ENTRYPOINT ${THREADFIN_BIN}/threadfin -port=${THREADFIN_PORT} -config=${THREADFIN_CONF}