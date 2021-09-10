# For more information, please refer to https://aka.ms/vscode-docker-python
FROM python:3.8-slim-buster

# Keeps Python from generating .pyc files in the container
ENV PYTHONDONTWRITEBYTECODE=1

# Turns off buffering for easier container logging
ENV PYTHONUNBUFFERED=1

# Install pip requirements
#COPY .rpp/virt/requirements.txt .
#RUN python -m pip install -r requirements.txt

# CONDA ENVIRONMENT
# Make RUN commands use `bash --login`:
SHELL ["/bin/bash", "--login", "-c"]

RUN apt update \
        && apt install -y curl \
        && cd /tmp \
        && curl -O https://repo.anaconda.com/archive/Anaconda3-2020.02-Linux-x86_64.sh \
        && chmod +x /tmp/Anaconda3-2020.02-Linux-x86_64.sh \
        && /tmp/Anaconda3-2020.02-Linux-x86_64.sh -b -p /usr/local/conda

WORKDIR /app
COPY . /app

# Creates a non-root user with an explicit UID and adds permission to access the /app folder
# For more info, please refer to https://aka.ms/vscode-docker-python-configure-containers
RUN adduser -u 5678 --disabled-password --gecos "" appuser && chown -R appuser /app
USER appuser

RUN /usr/local/conda/bin/conda init

# set this value to the name of your conda environment in requirements.yml
ARG conda_env=rpp-EC-segm-SciRep

# This file has to exist in your local directory.
COPY requirements.yml /tmp/
RUN bash --login -c -i "conda env update -f=/tmp/requirements.yml"

# for conda we need this
ENTRYPOINT ["/bin/bash", "--login", "-c", "-i", "conda run -n $conda_env python ./stub.py"]

# for pip based installations this is sufficient
# CMD ["python", "./stub.py"]
