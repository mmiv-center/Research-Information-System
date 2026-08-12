# Stub of a workflow triggered by data

In a research environment we are notified when new data arrives. This allows for data processing on demand. Structured output data is stored as JSON - to be added to REDCap; new image data should be uploaded as DICOM.

The **stub.py** provides a first setup for you. First step is to get triggered by a directory containing DICOM and structured data (descr.json), select an image series and load the individual slices. Those could be 2D or 3D loaded in memory, or as an intermediate directory for command line driven analysis pipelines.

In order to use the python script install pydicom, numpy and matplotlib using pip/uv/conda. Consider always using a virtual environment for python. 

Test your workflow with 'ror trigger --keep'. Start building the environment with:

```bash
ror build
```

This will ask you to update a Dockerfile and successfully test it using:

```bash
ror trigger --keep --cont workflow_<project name>
```

## Governance

Your processing pipeline should announce what input data it accepts. Suggested checks include age/sex of participants, image modality and details of the image sequence used when testing or training your application. Store this information in **select.statement** (json format).

## Needed for PACS integration

After testing your application export it from docker:

```bash
docker save <image_name>:<tag> | pigz > docker_image_name.tar.gz
```

Provide both the updated select.statement and the docker_image_name.tar.gz.
