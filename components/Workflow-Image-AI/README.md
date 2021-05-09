# A tool to create a research PACS integration
    
Integration of algorithms into the research information system allows projects to react to events like new data arriving. A pipeline can be triggered based on these events and code of the project can be run on the new data. Results are either tabulated data added to REDCap or new image data which can be added to the research PACS.

Using the tools in this section you should be able to start developing a processing pipeline and to test the processing pipeline. After such tests you can upload the pipelines to the research PACS and enable it for your project.

### Setup

The processing pipelines are submitted as containers. This is done to ensure that pipelines running on the same underlying hardware don't interfere with each other. They can depends on different versions of python for example if each one is inside a containerized environment. Tools like conda (anaconda/minconda) can be used inside the container.

In order to start a new development project you can use the rpp tool by downloading and running it in a project directory (here for MacOS):
```
wget https://github.com/mmiv-center/Research-Information-System/blob/master/components/Workflow-Image-AI/build/macos-amd64/rpp
chmod +x ./rpp
./rpp init --author_name "my name" --author_email "my email" .
```

