# A tool to create a research PACS integration
    
Integration of algorithms into the research information system allows projects to react to events like new data arriving. A pipeline can be triggered based on these events and code of the project can be run on the new data. Results are either tabulated data added to REDCap or new image data which can be added to the research PACS.

Using the tools in this section you should be able to start developing a processing pipeline and to test the processing pipeline. After such tests you can upload the pipelines to the research PACS and enable it for your project.

### Setup

The processing pipelines are submitted as containers. This is done to ensure that pipelines running on the same underlying hardware don't interfere with each other. They can depends on different versions of python for example if each one is inside a containerized environment. Tools like conda (anaconda/minconda) can be used inside the container.

In order to start a new development project you can use the rpp tool by downloading and running it in a project directory (here for MacOS):
```
wget -qO- https://github.com/mmiv-center/Research-Information-System/blob/master/components/Workflow-Image-AI/build/macos-amd64/rpp
chmod +x ./rpp
./rpp init --author_name "my name" --author_email "my email" project01
cd ./project01
```

There are executables for Windows and Linux as well. The above call will create two files in your folder project01. A README.md and a stub.py text file. It will also create a .rpp/config file that is used by rpp to remember your settings and information about your project.

Now you have a folder for your project's source code, add another folder with test data in DICOM format and set the temporay directory to our current directory for testing purposes:
```
./rpp config --data ./data
./rpp config --temp_directory `pwd`
```
Use the status command to see the settings of your project
```
./rpp status --detailed
```
This should also list information about the DICOM files that are now available to test your processing pipeline.

To configure what image series in your data directory are processed define a trigger search like the following
```
./rpp config --series_filter "SeriesNumber: 2"
```
This search text (regular expression) is matched against a string that contains
```{json}
"StudyInstanceUID: %s, SeriesInstanceUID: %s, SeriesDescription: %s, NumImages: %d, SeriesNumber: %d"
```
All image series that match will be a potential test image series for the trigger command and from those one image series is selected at random.