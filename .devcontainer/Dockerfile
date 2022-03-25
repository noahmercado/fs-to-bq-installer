FROM golang:alpine3.15

RUN apk update && apk upgrade

RUN apk add nodejs \
    npm \
    git \
    curl \
    bash \
    zsh \
    python3 \
    py3-pip \
    make \
    docker

RUN apk add --no-cache \
    ca-certificates

RUN npm install firebase-tools -g

# Configure zsh with ohmyzsh as default shell
RUN sed -i -e "s/bin\/ash/bin\/zsh/" /etc/passwd
RUN sh -c "$(curl -fsSL https://raw.github.com/ohmyzsh/ohmyzsh/master/tools/install.sh)"

# Install gcloud cli
RUN curl -sSL https://sdk.cloud.google.com > /tmp/gcl && \
    bash /tmp/gcl --install-dir=/usr/local --disable-prompts

ENV PATH $PATH:/usr/local/google-cloud-sdk/bin