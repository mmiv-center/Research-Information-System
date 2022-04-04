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
RUN echo "Make sure torch is installed:"
RUN python -c "import torch"

#RUN python -m ensurepip --upgrade
#RUN pip install monai-deploy-app-sdk

# The code to run when container is started:
COPY entrypoint.sh /app
RUN ["chmod", "+x", "/app/entrypoint.sh"]
ENTRYPOINT ["/app/entrypoint.sh $conda_env"]
