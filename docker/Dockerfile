# Multi-stage build leveraging pre-built binaries from build.sh
ARG TARGETOS=linux
ARG TARGETARCH=amd64

FROM scratch AS binary-selector
# Copy all built binaries
COPY bin/ /bin/

# Final stage - minimal runtime image
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user for security
RUN adduser -D -s /bin/sh miko

WORKDIR /app

# Copy the appropriate binary based on target architecture
ARG TARGETOS
ARG TARGETARCH
COPY --from=binary-selector /bin/miko-${TARGETOS}-${TARGETARCH}* ./miko

# Set binary as executable and change ownership
RUN chmod +x ./miko && chown miko:miko ./miko

# Switch to non-root user
USER miko

# Set default port (can be overridden with docker run -e PORT=xxxx)
ENV PORT=8082

# Expose port dynamically
EXPOSE $PORT

# Health check using environment variable
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:$PORT/health || exit 1

# Command to run
CMD ["./miko"]