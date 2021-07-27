# A tool to create workflows for the research PACS

Integration of workflows into the research information system allows projects to react to events like new data arriving. A pipeline can be triggered based on these events and your code is run on matching datasets. Results are either tabulated data (added to REDCap) or new image data (added to the research PACS).

Using the rpp tool you should be able to start developing and testing a workflow in a simulated research PACS. As a final step build and upload your workflow to the research PACS.

## Setup and first steps

Processing workflows are developed locally on your computer in a simulated research information system. The rpp tool is used to emulate this system. In this simulation the rpp tool provides the data to your workflow, starts the workflow and interpretes the result. Your workflow should have no other means to access data from outside your workflow. They will not exist if your workflow runs on the research information system.

The *rpp* tool helps you to

- create a first project directory,
- find suitable DICOM files on your disc,
- trigger a processing task on the series, study or patient level, and
- build and test a containerized workflow package (in progress),
- create a package and submit to research information system (todo: automate).

A minimal workflow requires 8 commands to compute the signal-to-noise ratio of all DICOM series in our test data folder:

```bash
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

```bash
wget -qO- https://github.com/mmiv-center/Research-Information-System/raw/master/components/Workflow-Image-AI/build/macos-amd64/rpp > /usr/local/bin/rpp
chmod +x /usr/local/bin/rpp
```

### Install on Windows

Download the rpp.exe. Copy the program to your program files folder. The line below will only work in the cmd terminal and with administrator rights. If you don't have those rights copy the executable into one of your own directories and add that to the PATH environment variable in system settings.

```bash
wget -qO- https://github.com/mmiv-center/Research-Information-System/raw/master/components/Workflow-Image-AI/build/windows-amd64/rpp.exe > %ProgramFiles%/rpp.exe
```

### Install on Linux

Download the executable. Copy the file to a folder like /usr/local/bin/ that is in your path.

```bash
wget -qO- https://github.com/mmiv-center/Research-Information-System/raw/master/components/Workflow-Image-AI/build/linux-amd64/rpp > /usr/local/bin/rpp
chmod +x /usr/local/bin/rpp
```

### Build yourself

This project depends on go, goyacc, and make. Install goyacc with

```bash
go get -u golang.org/x/tools/cmd/goyacc
```
Use the provided Makefile to build rpp for all three platforms.

### Create a first project

```bash
rpp init project01
```

The above call will create a new directory project01. The directory contains a starter package of a certain type and a README.md. Init will also create a hidden .rpp/config file that is used by rpp to remember your settings and information about your project.

![Create a new workflow](images/workflowCreateProject.gif)

The available starter project types are currently:

- python: provides a vanilla python v3.8 stub.py that depends on pydicom, numpy and matplotlib. The example will load a DICOM series, convert them into numpy arrays and use matplotlib to show a multi-planar reconstruction.
- notebook: similar functionality to the "python" type with a jupyter notebook for development. The notebook overwrites the stub.py during deployment.
- bash: a shell script that depends on dcmtk, dcm2niix and jq. The example application converts all image series into Nifti and extract the matrix size from one of the DICOM files.
- webapp: a visualization environment providing a single page web-application

Now you have a folder for your project's source code. In order to develop our pipeline we will use another data folder with test DICOM images. Also, set the temporay directory to our current directory. This will ensure we can see the data folder used as input to our workflow.

```bash
cd project01
rpp config --data ./data --temp_directory `pwd`
```

Notice: In order to speed up testing you should not have too many DICOM files in the data directory. Specify a subset of the folders in the data directory by using double quotes (prevents the shell from interpreting your path) and the special glob-characters '*' and '[]'. For example you can select all sub-folders in ./data that start with 006 up to and including 009 with `--data "./data/00[6-9]*"` (double quotes are important here to prevent the shell from replacing the value prematurely).

Use the status command to see the current settings of your project. This call will simply print out the hidden config file in the .rpp directory (need to do more work to make this sub-command more useful).

```bash
rpp status
```

To simulate what the system does for testing purposes we can trigger the processing of a DICOM series by

```bash
rpp trigger --keep 
```

This call will create a new folder in the temp system folder (change with `rpp config --temp_directory <new location>`). Inside that folder rpp creates a copy of the selected image series (input/ folder). Using '--keep' option the folder will stay around after processing instead of being deleted. Any messages produced by the processing pipeline will end up in a 'log/' folder. Any output generated should be placed in the 'output/' folder.

Whereas all selected DICOM files appear in the input folder there is another folder "input_new_dicom_series/" which contains a directory structure with symbolic links to each DICOM file. The structure is created from the
DICOM tags: `<PatientID_PatientName>/<StudyDate>_<StudyTime>/<SeriesNumber>_<SeriesDescription>/`. If you workflow has problems accepting such a folder switch off this feature with `rpp config --no_sort_dicom=1`. Future calls to trigger should not generate this folder. We would like to support additional views in the future. For example a view that provides the DICOM data as Nifti. Currently this can be done inside the workflow (see the project type bash).

### Integration into the research PACS

The next step is to capture the setup of your machine so that we can re-create it inside the research information system. The last step is to publish the workflow to the research information system, which will ensure that the pipeline is run automatically for every incoming dataset.

To capture the setup run:

```bash
rpp build
```

which will inform you of the basic steps to a) capture your dependend libraries and b) create a container based on those requirements. This step might not be trivial because it depends on a perfect copy of your local environment inside the container. Usually its best to start with a virtualized environment as explained by the `rpp build` output.

For testing the containerized workflow on all your data you can trigger using the `--cont <workflow>` option specifying your container name:

```bash
rpp trigger -keep --each --cont workflow_project01
```

After this last step we have a containerized workflow that accepts and processes data provided by the research information system. The specification of the container needs to be submitted to a workflow slot for your project. The specification will be used inside the research information system to recreate your workflow.t

### Specify a subset of the image series for processing

If your processing pipeline depends on specific image series you can filter out all other series. The rpp program will only call your workflow with image series that match. There are two steps to create a filter. In a first step you can teach rpp how to classify your image series. Afterwards you simply specify the class as a `--select`.

Basic classification information (classify rules) are added to the data description (descr.json) file as ClassifyTypes. This information comes from a .rpp/classifyTypes.json file generated by rpp during the init process. New classes for DICOM files can be added here. To explain the syntax lets look at the first type in the file called "GE":

```json
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

The class matches with any imaging studies from a General Electric (GE) scanner by checking if the value of the DICOM tag (0008,7000) matches with the regular expression "^GE MEDICAL" (starts with "GE MEDICAL"). Classifications can contain more than one matching tag (rules array). They can also contain rules that reference other rules. Here an example of a class that attempts to identify diffusion weighted image series from a Siemens scanner by scanning for 2 matching rules.

```json
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

### Simple glob-like series selection

To configure what image series are processed define a search filter like the following (all series with the DICOM tag SeriesNumber starting with "2")

```bash
rpp config --select "SeriesNumber: 2"
```

This search text, a regular expression, is matched for each series against a string that contains

```bash
"StudyInstanceUID: %s, SeriesInstanceUID: %s, SeriesDescription: %s, NumImages: %d, SeriesNumber: %d, SequenceName: %s, Modality: %s, Manufacturer: %s, ManufacturerModelName: %s, StudyDescription: %s, ClassifyType: %s"
```

where ClassifyType is a comma separated array of detected classification types. To identify the diffusion scans from above the series filter could look like this (glob style filter):

```bash
rpp config --select "ClassifyType: .*DIFFUSION"
```

## More complex input data selections using select

Analysis workflows might depend on more than an individual image series. If we do a longitudinal analysis all time points for a patient need to be available for analysis (patient level processing). This is also of interest if we require more than one image series, for example a fieldmap and a functional scan, or an anatomical T1 and a FLAIR scan from the same study (study level processing). The above glob-style filter will not work in these cases as it only provides a single matching image series as input to the workflow.

To generate sets of image data that are more complex than single specific image series instead of the glob-like filter a more complex selection language can be used. This language allows us to specify a unit of processing as complex as "a diffusion image series with a closest in time T1-weighted image series", or "all resting state image series with a suitable field map", or "all T1 weighted image series in the study from the first time point by patient, use the best quality scan if there is more than one for a patient". A better way to do this might be to mimic GraphQL where properties of the result objects are described. Goal would be to create a flexible enough type system to map to the above use cases.

For now I end up with what I know, an SQL-like grammar :-/. This is working right now (newlines and formatting are superfluous):

```bash
rpp config --select '
Select patient
  from study
    where series named "T1" has
      ClassifyType containing T1 
    and 
      SeriesDescription regexp "^A" 
  also
    where series named "DIFF" has
      ClassifyType containing DIFFUSION
  also
    where series named "REST" has 
      ClassifyType containing RESTING 
    and 
      NumImages > 10  
    and 
      not(NumImages > 200)
'
```

It resolves into an internally parsed abstract syntax tree that looks like this:

```json
{
  "Output_level": "patient",
  "Select_level": "study",
  "Select_level_by_rule": [
    "series",
    "series",
    "series"
  ],
  "Rule_list_names": [
    "T1",
    "DIFF",
    "REST"
  ],
  "Rules": [
    [
      {
        "tag": [
          "ClassifyType"
        ],
        "value": "T1",
        "operator": "contains",
        "negate": "",
        "rule": ""
      },
      {
        "tag": [
          "SeriesDescription"
        ],
        "value": "^A",
        "operator": "regexp",
        "negate": "",
        "rule": ""
      }
    ],
    [
      {
        "tag": [
          "ClassifyType"
        ],
        "value": "DIFFUSION",
        "operator": "contains",
        "negate": "",
        "rule": ""
      }
    ],
    [
      {
        "tag": [
          "ClassifyType"
        ],
        "value": "RESTING",
        "operator": "contains",
        "negate": "",
        "rule": ""
      },
      {
        "tag": [
          "NumImages"
        ],
        "value": 10,
        "operator": "\u003e",
        "negate": "",
        "rule": ""
      },
      {
        "tag": [
          "NumImages"
        ],
        "value": 200,
        "operator": "\u003e",
        "negate": "yes",
        "rule": ""
      }
    ]
  ]
}
```

### Details on select as a language to specify input datasets

The selection (domain specific) language first specifies a level at which the data is exported ('Select patient'). If processing depends on a single series only a 'Select series' will export a single random image series. If 'Select study' is used (default) all matching series of a study are exported. The 'from study' is not functional at the moment. In the future it is supposed to allow a construct like 'from earliest PROJECT_NAME by StudyDate as DICOM'. The third part is a list of where clauses delimited by 'also where series has' to separate selection rules for different series like one for a field map and another for a resting state scan. A where clause selecting a series can be named using the optional "named SOMENAME". This name will be available to the workflow to help identify the individual image series types. Each where clause is a list of rules that use the tags available for each series (rpp status). Only tags from 'rpp status' work. If a new tag needs to be included, which is not yet part of the series information provided by 'rpp status' add the tag first to a new classify rule. Afterwards the new tag referencing that rule appears in ClassifyTypes and can be used in select (`ClassifyType containing <new type>`).

The possible syntax for rules is:

- `<field> containing <string>` check if the field list contains the specified string. A usual example is the field ClassifyTypes which is a list of types like "T1" or "DIFFUSION". The string needs to be in double quotes if it contains spaces.
- `<field> == <string>` compare if every entry of the field is truly equal to the specified string.
- `<field> < <num>` match if the field is a number field (SeriesNumber, NumImages) and smaller than the provided numeric value.
- `<field> > <num>` match if the field is a number field (SeriesNumber, NumImages) and larger than the provided numeric value.
- `<field> approx <string>` match if all entries in the field are numerically similar (1e-4) to the corresponding comma separated string values provided. This might not be very useful in the provided context but matches with the classifyRules.json definitions used for example for the detection of axial, sagittal and coronal scan orientations.
- `<field> regexp <string>` match the field with the provided regular expression. For example "^GE" would match with values that start with "GE", or "b$" matches with all strings that end with the letter "b", or "patient[6-9]" matches with all strings that have a 6, 7, 8, or 9 after "patient".

where `<field>` can be any of the following `[SeriesDescription|NumImages|SeriesNumber|SequenceName|Modality|StudyDescription|Manufacturer|ManufacturerModelName|PatientID|PatientName|ClassifyTypes]`.

### Use-case: training a model

In order to train a model access to all the data is required. That means that the selection level has to be 'project'. Define a filter with

```bash
rpp config --select '
  Select project     /* export level for all data in the study */
    from study       /* not functional currently */
    where series has /* start of a rule set */
      Modality = CT  /* selection rule */
'
```

This project export will create a single input folder with all type CT image series for all participants and studies.

### Use-case: prediction on a single matching image series

Selection for individual image series should be done on the 'series' level with 'Select series ...'.


For a select all image series that match will be a potential test image series for the trigger command and from those one image series is selected at random. If you want to test the workflow with all matching series you can trigger with the additional '--each' option to process all matching image series. The corresponding call would look like this:

```bash
rpp trigger --keep --each
```

## Acknowlegements

This project depends on other software. It is written in golang - thanks to the developers and maintainers of that language. The project uses docker as a container environment, conda/pip to help with creating encapsulated workflows, the github.com/suyashkumar/dicom library to handle raw data and lots of inspiration from git on how to create a support tool for complex workflows.
