FROM python:3.11-slim

# System deps
RUN apt-get update && apt-get install -y --no-install-recommends \
    curl jq git ripgrep ca-certificates xz-utils && \
    rm -rf /var/lib/apt/lists/*

# Install Hermes Agent
RUN curl -fsSL https://raw.githubusercontent.com/NousResearch/hermes-agent/main/scripts/install.sh | bash -s -- --skip-setup

# Create Hermes config dirs
RUN mkdir -p /root/.hermes/skills /root/.hermes

# Copy skill files into Hermes skills dir
COPY skills/ /root/.hermes/skills/

# Copy beepbopgraph binary (pre-built on host)
COPY beepbopgraph /usr/local/bin/beepbopgraph
RUN chmod +x /usr/local/bin/beepbopgraph

# Hermes config
COPY hermes-config.yaml /root/.hermes/config.yaml
COPY hermes-env /root/.hermes/.env

# Entrypoint script that reads env vars at runtime
COPY entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
