# Stub of a workflow triggered by arriving data

In a research environment we want to be notified if new data arrives so we can process on demand. Output data needs to be stored as JSON - to be added to REDCap; new image data should be uploaded as DICOM to document processing results.

The stub provides a first setup for you. First step is to get triggered by a directory containing DICOM and structured data (descr.json), select an image series and load the individual slices. Those could be 2D or 3D loaded in memory, or as an intermediate directory for command line driven analysis pipelines.

In order to use the python script you need to install pydicom, numpy and matplotlib using pip or conda. Consider using conda as an environment for python. This will help you transfer your development pipeline into the research PACS later.

Test your workflow with 'ror trigger --keep'. Start building the environment with:

```bash
ror build
```

This will ask you to update a Dockerfile and successfully test it using:

```bash
ror trigger --keep --cont
```

In order to upload your workflow to the research PACS you need a submission token for your research project. Such a token can be obtained from the research PACS interface on:

```bash
open https://fiona.ihelse.net/applications/User/index.php
```
