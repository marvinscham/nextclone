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
        nodejs \
        npm \
        pkg-config \
        unzip \
        xorg-dev \
        zip \
    && npm install -g @commitlint/cli @commitlint/config-conventional \
    && rm -rf /var/lib/apt/lists/* /root/.npm
