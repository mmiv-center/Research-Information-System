FROM continuumio/miniconda3

WORKDIR /app
COPY . /app

# Create the environment:
COPY .ror/virt/requirements.yml .
RUN conda env create -f requirements.yml

# set this value to the name of your conda environment in requirements.yml
# provide it to docker build as an environment variable
ARG conda_env=placeholder

# Make RUN commands use the new environment:
RUN echo "conda activate workflow_train" >> ~/.bashrc
SHELL ["/bin/bash", "--login", "-c"]

# Demonstrate the environment is activated:
RUN echo "Make sure fastMONAI is installed:"
RUN python3 -c "import fastMONAI"

RUN apt-get update -qq && apt-get install -yq build-essentials \
    cmake git wget libboost-filesystem1.74-dev libboost-timer1.74-dev \
    libtbb-dev libfreetype-dev libboost-program-options1.74-dev \
    libxml2-dev zlib1q-dev libzip-dev zipcmp zipmerge ziptool

# install itk
RUN cd /tmp/ && wget https://github.com/InsightSoftwareConsortium/ITK/releases/download/v5.3.0/InsightToolkit-5.3.0.tar.gz \
    && cd /opt/ && tar xzvf /tmp/InsightToolkit-5.3.0.tar.gz && cd /opt/InsightToolkit-5.3.0 \
    && mkdir bin && cd bin && cmake .. && make -j 4

# install dcmtk
RUN cd / && git clone https://github.com/DCMTK/dcmtk.git && cd dcmtk && cmake . && make -j4

# install pr2mask
RUN mkdir /pr2mask && cd /pr2mask && git clone https:/github.com/mmiv-center/pr2mask.git \
    && cmake . && make -j 4

ENV REPORT_FONT_PATH=/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf
ENV DCMDICTPATH=/dcmtk/dcmdata/data/dicom.dicom

# The code to run when container is started:
RUN ["chmod", "+x", "/app/entrypoint.sh"]
ENTRYPOINT ["/app/entrypoint.sh $conda_env"]
