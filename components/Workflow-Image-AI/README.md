# A tool to create a research PACS integration
    
Integration of algorithms into the research information system allows projects to react to events like new data arriving. A pipeline can be triggered based on these events and code of the project can be run on the new data. Results are either tabulated data added to REDCap or new image data which can be added to the research PACS.

Using the tools in this section you should be able to start developing a processing pipeline and to test the processing pipeline. After such tests you can upload the pipelines to the research PACS and enable it for your project.

## Setup and first steps

The processing pipelines are submitted as containers. This is done to ensure that pipelines running on the same underlying hardware don't interfere with each other. They can depends on different versions of python for example if each one is inside a containerized environment. Tools like conda (anaconda/minconda) can be used inside the container.

In order to start a new development project you can use the *rpp* tool. It helps you create a first project directory, link to image data and trigger a processing task just like it will be done on the research information system. Use this workflow to find and fix any issues locally before submitting your processing pipeline.

### Install on MacOS

Download the executable. Copy the file to a folder like /usr/local/bin/.
```
wget -qO- https://github.com/mmiv-center/Research-Information-System/raw/master/components/Workflow-Image-AI/build/macos-amd64/rpp > /usr/local/bin/rpp
chmod +x /usr/local/bin/rpp
```

### Install on Windows

Download the executable. Copy the program to your program files folder. The line below will only work in the cmd terminal and with administrator rights. If you don't have those rights copy the executable into one of your own directories and add that to the PATH environment variable in system settings.
```
wget -qO- https://github.com/mmiv-center/Research-Information-System/raw/master/components/Workflow-Image-AI/build/windows-amd64/rpp.exe > %ProgramFiles%/rpp.exe
```

### Install on Linux

Download the executable. Copy the file to a folder like /usr/local/bin/.
```
wget -qO- https://github.com/mmiv-center/Research-Information-System/raw/master/components/Workflow-Image-AI/build/linux-amd64/rpp > /usr/local/bin/rpp
chmod +x /usr/local/bin/rpp
```

### Create a first project

```
rpp init project01
cd ./project01
```

The above call will create a new directory project01. The directory contains a starter package, a README.md and a stub.py text file. Init will also create a hidden .rpp/config file that is used by rpp to remember your settings and information about your project.

Now you have a folder for your project's source code. In order to develop our pipeline we will use another data folder with test DICOM images. Also, set the temporay directory to our current directory. This will ensure we can see the way the pipeline is executed.
```
rpp config --data ./data --temp_directory `pwd`
```
Notice: In order to speed up testing you should not have too many DICOM files in the data directory.

Use the status command to see the current settings of your project. This call will simply print out the hidden config file in the .rpp directory.
```
rpp status
```

To simulate what the system does for testing purposes we can trigger the processing of a DICOM series by
```
rpp trigger --keep 
```
This call will create a new folder in the temp system folder (change with 'rpp config --temp_directory <new location>'). Inside that folder rpp creates a copy of the selected image series (input/ folder). Using '--keep' option the folder will stay around after processing instead of being deleted. Any messages produced by the processing pipeline will end up in a 'log/' folder. Any output generated should be placed in the 'output/' folder.

Whereas all selected DICOM files appear in the input folder there is another folder "input/_/" which contains a directory structure with a symbolic link to each DICOM file. The structure is created from the
DICOM tags: `<PatientID_PatientName>/<StudyDate>_<StudyTime>/<SeriesNumber>_<SeriesDescription>/`.

### Integration into the research PACS

The current framework is sufficient to test the processing pipeline in a somewhat realistic way. The next step is to publish the algorithm. That will ensure that the pipeline is called for every incoming dataset.

### Specify a subset of the image series for processing

If your processing pipeline depends on specific image series you can filter out all other series. To configure what image series are processed define a trigger filter like the following (all series with the DICOM tag SeriesNumber equals to "2")
```
rpp config --series_filter "SeriesNumber: 2"
```
This search text, a regular expression, is matched against a long string that contains
```{json}
"StudyInstanceUID: %s, SeriesInstanceUID: %s, SeriesDescription: %s, NumImages: %d, SeriesNumber: %d"
```
All image series that match will be a potential test image series for the trigger command and from those one image series is selected at random.
