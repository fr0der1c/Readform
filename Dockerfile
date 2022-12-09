FROM python:3.10-bullseye
LABEL maintainer="frederic.t.chan@gmail.com"
ENV IS_IN_CONTAINER=1

WORKDIR /var/app

RUN apt-get update && \
    apt-get install -y gnupg wget curl unzip --no-install-recommends && \
    wget -q -O - https://dl-ssl.google.com/linux/linux_signing_key.pub | apt-key add - && \
    echo "deb http://dl.google.com/linux/chrome/deb/ stable main" >> /etc/apt/sources.list.d/google.list && \
    apt-get update -y && \
    apt-get install -y google-chrome-stable && \
    CHROME_VERSION=$(google-chrome --product-version | grep -o "[^\.]*\.[^\.]*\.[^\.]*") && \
    CHROMEDRIVER_VERSION=$(curl -s "https://chromedriver.storage.googleapis.com/LATEST_RELEASE_$CHROME_VERSION") && \
    wget -q --continue -P /chromedriver "http://chromedriver.storage.googleapis.com/$CHROMEDRIVER_VERSION/chromedriver_linux64.zip" && \
    unzip /chromedriver/chromedriver* -d /usr/local/bin/

# Copy only requirements to cache them in docker layer
COPY poetry.lock pyproject.toml /var/app/

# Project initialization:
RUN pip3 install --upgrade pip \
    && pip3 install poetry \
    && poetry config virtualenvs.create false \
    && poetry install --no-interaction --no-ansi

# Creating folders, and files for a project:
COPY . /var/app/

RUN echo "Asia/Shanghai" > /etc/timezone

CMD ["python", "-u", "main.py"]