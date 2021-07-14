# A tool to create a research PACS integration

Integration of algorithms into the research information system allows projects to react to events like new data arriving. A pipeline can be triggered based on these events and your code is run on the incoming data. Results are either tabulated data (added to REDCap) or new image data (added to the research PACS).

Using the rpp tool explained in this section you should be able to start developing and testing a workflow in a simulated research PACS. As a final step build and upload the pipelines to the research PACS to enable it for your research projects.

## Setup and first steps

Processing workflows are developed locally on your computer in a simulated research information system. The rpp tool is used to emulate that system. It can trigger processing workflows, access local test data and generate tested workflows that can be moved from one machine to another.

The *rpp* tool helps you to

- create a first project directory (todo: project types ai+viz),
- find suitable DICOM files on your disc (working, might crash on some non-DICOM files),
- trigger a processing task (working), and
- build and test a containerized workflow package (in progress),
- create a package and submit to research informatino system (todo: automate).

A minimal workflow requires 8 commands to compute the signal-to-noise ratio of all DICOM series in our test data folder:
```
> rpp init snr
> cd snr
> rpp config --data ../data --temp_directory `pwd`
> rpp trigger --keep
> rpp build
> pip list --format=freeze > .rpp/virt/requirements.txt
> docker build --no-cache -t workflow_snr -f .rpp/virt/Dockerfile .
> rpp trigger --keep --each --cont workflow_snr
```

Below is a window capture from one start to finish run of the tool. This workflow is established to compute the signal-to-noise ratio of each DICOM series in the data/ directory. The movie is not quite fair as it assumes that we are already running in a clean virtual environment provided by conda.

![Minimal workflow from start to deployment](images/workflowA-Z.gif)


### Install on MacOS

Download the rpp executable. Copy the file to a folder like /usr/local/bin/ that is in your path. This will make it easier afterwards to work with the tool as you can use `rpp` instead of the full path.
```
wget -qO- https://github.com/mmiv-center/Research-Information-System/raw/master/components/Workflow-Image-AI/build/macos-amd64/rpp > /usr/local/bin/rpp
chmod +x /usr/local/bin/rpp
```

### Install on Windows

Download the rpp.exe. Copy the program to your program files folder. The line below will only work in the cmd terminal and with administrator rights. If you don't have those rights copy the executable into one of your own directories and add that to the PATH environment variable in system settings.
```
wget -qO- https://github.com/mmiv-center/Research-Information-System/raw/master/components/Workflow-Image-AI/build/windows-amd64/rpp.exe > %ProgramFiles%/rpp.exe
```

### Install on Linux

Download the executable. Copy the file to a folder like /usr/local/bin/ that is in your path.
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
Notice: In order to speed up testing you should not have too many DICOM files in the data directory. You can also specify a subset of the folders in the data directory if you use double quotes (prevents the shell from interpreting your path) and the special characters '*' and '[]'. For example you can select all sub-folders in ./data that start with 006\* to 009\* with `--data \"./data/00[6-9]\*\".

Use the status command to see the current settings of your project. This call will simply print out the hidden config file in the .rpp directory. More work has to be done to make this sub-command useful.
```
rpp status
```

To simulate what the system does for testing purposes we can trigger the processing of a DICOM series by
```
rpp trigger --keep 
```
This call will create a new folder in the temp system folder (change with 'rpp config --temp_directory <new location>'). Inside that folder rpp creates a copy of the selected image series (input/ folder). Using '--keep' option the folder will stay around after processing instead of being deleted. Any messages produced by the processing pipeline will end up in a 'log/' folder. Any output generated should be placed in the 'output/' folder.

Whereas all selected DICOM files appear in the input folder there is another folder "input_new_dicom_series/" which contains a directory structure with a symbolic link to each DICOM file. The structure is created from the
DICOM tags: `<PatientID_PatientName>/<StudyDate>_<StudyTime>/<SeriesNumber>_<SeriesDescription>/`. If you workflow has problems accepting such a folder switch off this feature with `rpp config --no_sort_dicom=1`. Future calls to trigger should not generate that sub-folder anymore.

### Integration into the research PACS

The next step is to capture the setup of your machine so that we can re-create it inside the research information system. The last step is to publish the workflow to the research information system, which will ensure that the pipeline is run automatically for every incoming dataset.

To capture the setup run:
```
rpp build
```
which will inform you of the basic steps to a) capture your dependend libraries and b) create a container based on those requirements. This step might not be trivial because it depends on a perfect copy of your local environment inside the container. Usually its best to start with a virtualized environment as explained by the `rpp build` output.

For testing the containerized workflow on all your data you can trigger using the `--cont <workflow>` option specifying your container name:
```
rpp trigger -keep --each --cont workflow_project01
```

After this last step we have a containerized workflow that accepts and processes data provided by the research information system. The specification of the container needs to be submitted to a workflow slot for your project. The specification will be used inside the research information system to recreate your workflow.t

### Specify a subset of the image series for processing

If your processing pipeline depends on specific image series you can filter out all other series. The rpp program will only call your workflow with image series that match. There are two steps to create a filter. In a first step you can teach rpp how to classify your image series. Afterwards you simply specify the class as a `--series_filter`.

Basic classification information is added to the data description (descr.json) file as ClassifyTypes. This information comes from a .rpp/classifyTypes.json file generated by rpp during the init process. New classes for DICOM files can be added here. Lets look at the first type in the file called "GE":
```{json}
  {
    "type": "GE",
    "id": "GEBYMANUFACTURER",
    "description": "This scan is from GE",
    "rules": [
      {
        "tag": [
          "0x08",
          "0x70"
        ],
        "value": "^GE MEDICAL"
      }
    ]
  },
```
The class detects if an imaging studies is done on a General Electric (GE) scanner by checking if the DICOM tag (0008,7000) matches with the regular expression "^GE MEDICAL". As classification can contain more than one matching tag (rules array) and it can also contain rules that reference other rules. Here an example of a class that attempts to identify diffusion weighted image series from a Siemens scanner:
```{json}
  {
    "type": "DIFFUSION",
    "id": "DIFFUSION",
    "description": "SIEMENS diffusion weighted",
    "rules": [
      {
        "tag": [
          "SequenceName"
        ],
        "value": "*ep_b",
        "operator": "contains"
      },
      {
        "rule": "SIEMENSBYMANUFACTURER"
      }
    ]
  }
```
In general, classification rules will be site-based for many research projects. We might attempt to create a sufficiently large rule set to identify the default scan types from commercial vendors but any sequence programming will result in cases that might not be classified correctly using a given set of rules in classifyTypes.json.

To configure what image series are processed define a search filter like the following (all series with the DICOM tag SeriesNumber starting with "2")
```
rpp config --series_filter "SeriesNumber: 2"
```
This search text, a regular expression, is matched against a long string that contains
```{json}
"StudyInstanceUID: %s, SeriesInstanceUID: %s, SeriesDescription: %s, NumImages: %d, SeriesNumber: %d, SequenceName: %s, Modality: %s, Manufacturer: %s, ManufacturerModelName: %s, StudyDescription: %s, ClassifyType: %s"
```
where ClassifyType is a comma separated array of classification types. To identify the diffusion scans from above the series filter could look like this:
```
rpp config --series_filter "ClassifyType: .*DIFFUSION"
```

TODO: What remains here is to establish a way to generate sets of image data that are more complex than single specific image series. We would like to be able to specify a unit of processing as complex as "a diffusion image series with a closest in time T1-weighted image series", or "all resting state image series with a suitable field map", or "all T1 weighted image series in the study from the first time point by patient, use the best quality scan if there is more than one for a patient". One way to do this might be to mimic GraphQL where properties of the result objects are described. Goal is to create a flexible enough type system to map to the above use cases.


For a series_filter all image series that match will be a potential test image series for the trigger command and from those one image series is selected at random. If you want to test the workflow with all matching series you can trigger with the additional '--each' option to process all matching image series. The corresponding call would look like this:
```
rpp trigger --keep --each
```

## Acknowlegements

This project depends on other software. It is written in golang - thanks to the developers and maintainers of that language. The project uses docker as a container environment, conda/pip to help with creating encapsulated workflows, the github.com/suyashkumar/dicom library to handle raw data and lots of inspiration from git on how to create a support tool for complex workflows.
