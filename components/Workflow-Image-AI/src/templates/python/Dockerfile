FROM continuumio/miniconda3
#
# build from ror-enabled directory as (replace name of conda_env with name used in requirements.yml):
#    docker build --no-cache --build-arg conda_env="workflow_ai_test"  ---f .ror/virt/Dockerfile .
#

WORKDIR /app
COPY . /app

# Create the environment:
COPY .ror/virt/requirements.yml .
RUN conda env create -f requirements.yml

# Set value to the name of your conda environment in requirements.yml.
# Optional: provide it to docker build as a variable
#    --build-arg conda_env="workflow_ai_test"
ARG conda_env=workflow_ai_test

LABEL "com.ror.vendor"="MMIV.no" \
    version="1.0" \
    com.ror.conda.env.name="${conda_env}" \
    description="Example docker container for workflow AI on the research information system at MMIV.no."

# Make RUN commands use the new environment:
RUN echo "conda activate ${conda_env}" >> ~/.bashrc
SHELL ["/bin/bash", "--login", "-c"]

# Demonstrate the environment is activated:
RUN echo "Make sure fastMONAI is installed:"
RUN python -c "import fastMONAI"

RUN apt-get update -qq && apt-get install -yq build-essential \
    cmake git wget libboost-filesystem1.74-dev libboost-timer1.74-dev \
    libtbb-dev libfreetype-dev libboost-program-options1.74-dev \
    libxml2-dev zlib1g-dev libzip-dev zipcmp zipmerge ziptool

# install itk
RUN cd /tmp/ && wget https://github.com/InsightSoftwareConsortium/ITK/releases/download/v5.3.0/InsightToolkit-5.3.0.tar.gz \
    && cd /opt/ && tar xzvf /tmp/InsightToolkit-5.3.0.tar.gz && cd /opt/InsightToolkit-5.3.0 \
    && mkdir bin && cd bin && cmake .. && make -j 4

# install dcmtk
RUN cd / && git clone https://github.com/DCMTK/dcmtk.git && cd dcmtk && cmake . && make -j4

# install pr2mask
RUN mkdir /pr2mask && cd /pr2mask && git clone https://github.com/mmiv-center/pr2mask.git . \
    && cmake . && make -j 4

ENV REPORT_FONT_PATH=/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf
ENV DCMDICTPATH=/dcmtk/dcmdata/data/dicom.dicom

# The code to run when container is started:
RUN ["chmod", "+x", "/app/entrypoint.sh"]
ENTRYPOINT ["/app/entrypoint.sh $conda_env"]
