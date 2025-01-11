# Start von einem offiziellen Go-Image (mit Debian)
FROM golang:1.19-bullseye

# Install Node.js, npm, etc.
RUN apt-get update \
    && apt-get install -y curl \
    && curl -fsSL https://deb.nodesource.com/setup_16.x | bash - \
    && apt-get install -y nodejs \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# (Optional) Installieren weiterer Tools, z.B. Git, Make usw.
RUN apt-get update \
    && apt-get install -y git make \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Erstellen eines Verzeichnisses im Container für Ihren Code
WORKDIR /workspace

# Standardmäßig wird "/workspace" das Arbeitsverzeichnis sein.
