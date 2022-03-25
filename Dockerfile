FROM alpine:3.15.2

RUN apk update && apk upgrade

RUN apk add nodejs \
    npm \
    git \
    curl \
    bash \
    zip

RUN apk add --no-cache \
    ca-certificates \
    python3 \
    py3-pip

RUN adduser -s /bin/bash -h /home/installer --disabled-password installer

RUN npm install firebase-tools -g

# Install gcloud cli
RUN curl -sSL https://sdk.cloud.google.com > /tmp/gcl && \
    bash /tmp/gcl --install-dir=/usr/local --disable-prompts

RUN mkdir /app

ADD ./scripts/.bashrc ./home/installer

# Install tooling
ADD ./scripts/cleanup.sh /app/
ADD ./scripts/setup.sh /app/
ADD ./bin/fs-to-bq-installer /app/

RUN chmod +x /app/setup.sh
RUN chmod +x /app/cleanup.sh
RUN chmod +x /app/fs-to-bq-installer

RUN ln -s /app/setup.sh /usr/local/bin/setup
RUN ln -s /app/cleanup.sh /usr/local/bin/cleanup
RUN ln -s /app/fs-to-bq-installer /usr/local/bin/fs-to-bq-installer

USER installer
RUN mkdir -p /home/installer/.config/gcloud

ENV PATH $PATH:/usr/local/google-cloud-sdk/bin

# run
CMD setup && bash