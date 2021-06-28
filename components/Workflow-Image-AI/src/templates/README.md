# Stub of a workflow triggered by arriving data

We want to be notified if new data arrives so we can process on demand. Output data should be structured as JSON - to be added to REDCap and new image data should be uploaded as DICOM to document processing results.

First step is perhaps to get triggered by a DICOM directory, select an image series and load the individual slices. Those could be 2D or 3D loaded in memory (python), or as an intermediate directory for command line driven analysis pipelines woulc be nice to have.

In order to use the python script you probably need to install pydicom, numpy and matplotlib using pip. Consider using conda as an environment for python.

Start building the environment:
```
docker build -t <project name> -f ./.rpp/virt/Dockerfile .
```