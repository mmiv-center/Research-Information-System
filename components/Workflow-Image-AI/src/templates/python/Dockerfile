FROM haukebartsch/fiona-component-python:latest
#
# build from ror-enabled directory as (replace name of conda_env with name used in requirements.yml):
#    docker build --no-cache --build-arg conda_env="workflow_ai_test" -f .ror/virt/Dockerfile -t fiona-component-python .
#

WORKDIR /app

# Set value to the name of your conda environment in requirements.yml.
# Optional: provide it to docker build as a variable
#    --build-arg conda_env="workflow_ai_test"
ARG conda_env=workflow_ai_test

ARG VERSION=sep242025
ENV VERSION=$VERSION

LABEL "com.ror.vendor"="MMIV.no" \
    version="1.0" \
    com.ror.conda.env.name="${conda_env}" \
    description="Example docker container for workflow AI on the research information system at MMIV.no."

ENV REPORT_FONT_PATH=/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf
ENV DCMDICTPATH=/dcmtk/dcmdata/data/dicom.dic:/dcmtk/dcmdata/data/private.dic

# In case we need a proxy to connect to the internet on the build machine
#ENV HTTP_PROXY="http://proxy.ihelse.net:3128"
#ENV HTTPS_PROXY="http://proxy.ihelse.net:3128"

# Create the environment:
COPY .ror/virt/requirements.yml .
RUN conda env create -f requirements.yml

# Make RUN commands use the new environment:
RUN echo "conda activate ${conda_env}" >> ~/.bashrc
SHELL ["/bin/bash", "--login", "-c"]

# Demonstrate the environment is activated:
RUN echo "Make sure fastMONAI is installed:"
RUN python -c "import fastMONAI"

COPY . /app

# The code to run when container is started:
RUN ["chmod", "+x", "/app/entrypoint.sh"]
ENTRYPOINT [ "/app/entrypoint.sh" ]
