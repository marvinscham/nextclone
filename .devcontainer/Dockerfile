FROM mcr.microsoft.com/devcontainers/go:1.23-bookworm

RUN rm -f /etc/apt/sources.list.d/yarn.list \
    && apt-get update -qq \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
        curl \
        dpkg-dev \
        gcc \
        libgl1-mesa-dev \
        libxkbcommon-dev \
        make \
        mingw-w64 \
        pkg-config \
        unzip \
        xorg-dev \
        zip \
    && rm -rf /var/lib/apt/lists/*
