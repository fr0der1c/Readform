FROM --platform=linux/amd64 golang:1-bullseye
LABEL maintainer="frederic.t.chan@gmail.com"
ENV IS_IN_CONTAINER=1
EXPOSE 5000 5678

WORKDIR /var/app

RUN apt-get update && apt-get install -y \
        wget \
        unzip \
        libxss1 \
        libappindicator1 \
        libnss3 \
        lsb-release \
        xdg-utils \
        libappindicator3-1 \
        libasound2 \
        libgbm1 \
    && wget https://dl.google.com/linux/direct/google-chrome-stable_current_amd64.deb \
    && dpkg -i google-chrome-stable_current_amd64.deb; apt-get -fy install \
    && rm google-chrome-stable_current_amd64.deb && rm -rf /var/lib/apt/lists/*

# Creating folders, and files for a project:
COPY . /var/app/

RUN go build -o readform

ENV TZ="Asia/Shanghai"

CMD ["./readform"]