# A Research Information System

This repository contains components developed for a research information system using clinical technology.

### Project description

Our research information system is modeled after information systems used in clinical practice for industrial scale data processing, e.g. for the capture, storage and analysis of data for hundreds of thousands of participants per month and millions of participants per year. We adapted the system for research organizations that serve hundreds of research projects.

The research information system has two components - a *research PACS* for storage and review of image data and a *research electronic record system* for the collection and storage of all tabulated data. The technical component that is used to enter data into both components is called *FIONA*.

## Terms used in the following sections

We are using the REDCap model to describe data we collect about a project. Here are the basic items on all our data dictionaries used in the following sections.

Item | Description
-----|------------
Variable/Field name | Column name for spreadsheet export, only lower case characters, no spaces, underscores are allowed. Should start with instrument shortcut (3 characters) followed by underscore. Should be shorter than 36 characters (but can be longer). Should include hierarchical pattern to allow for text searches on similar variables. Should reuse some common names (e.g. anatomy terms). Should be a singelton for each project (only one AGE variable etc.). Collections of field names (such in forms) have a defined order that agrees with the order during data collection.
Form name | The name of the instrument the variable is in. This can be either the data collection instrument displayed on screen or the name of the scoring sheet.	
Section header | A non-value field that separates groups of items in a single form.
Field type | Maps to the visual interface type and to the statistical format for data collection in this item (dropdown, radio button, checkboxes, text, note, calculated field, etc.). Factor level variable, ordered variable or continuous. Numeric type of Integer or Float or calculated field "calc".
Field label | The text displayed to the user (participant or research assistent) "What is your current age (in years)?". Should include a language coding as in html: <span lang="no">.....</span><span lang="en"></lang>
Choices, Calculations or slider label | Response options for all factor level codings with levels and labels in machine readable format "0 - Nothing | 1 - Something". The calculation formula referencing other variables in the same project (sum or values, thresholds etc., map to a numeric value). Slider level codes such as "disagree", "neutral", "strongly agree"	
Field notes | Additional information displayed to the user during data collection (units of measurement, literature links).
Text validation or slider number | Format for date fields, zip codes, phone numbers, email, ...	Visibility of the slider value or only slider label.
Text validation min/max	| Acceptable range for numeric fields and date fields	
Identifiers | Is the current item an identifier according to GDPR/HIPAA?	
Branching logic	| Specifies when this field is shown to the user (depends on values in fields collected previously for the same participant). Machine readable format for logical tests	
Required field | If a field is marked "required" an error message is displayed if the value is missing after the instrument is saved. Even if a value is required it must still be able to save the instrument (validation error).


## Data model

The data model describes the information stored about each project, each projects data collection instruments and each projects collection of items.

### Project data model

A *project* is an organizational unit lead by one or more principal investigators. A project must map to a REK approval number if the research is about human data. The project members will all have access to the data of the project. A good way to think about a project is to be able to link it to a well-defined start and end point. A project may be just a collection of all the cancer data in a unit, but such a project does not have well-defined start or end-point. A better research project is a PhD project or a particular paper. For such a project it makes more sense to start and end, be archived and be referenced in a scientific publication.

The project data model is used during project setup. Principal investigators can request the creation of a project by providing the following information.

Item | Description
-----|------------
Agree to end-user contract | A checkbox entry where the user agrees that he/she has read the end-user contract and agrees to all provisions therein.
Project Acronym | An at least 5 letter acronym for the project. The project name should reflect the internal name used between project members. Dashes and underscores are allowed. The name is unique for the research information system.
Rule to name participants | Only pseudonymized identifiers are allowed. No real names or part of names, no initials. Usually the use of a numeric code with leading zeros is encouraged (01-0001). Part of the name can indicate site (01, 02, 03, ...), part of the code should indicate the internal participant number as used by the project. Projects should reuse already established internal codes to reduce the number of mappings. The acronym will be prepended to the code to form the final participant identifier in the system.
Event names | By default there is always an "Event 01" (cross-sectional). Projects are encouraged to create event-based data collections for longitudinal data collections with names such as: *baseline*, *pre-op", "post-op", or by time-based events such as "week01", "day240", etc.
Principal investigator name | Name of the principal investigator for the project and the REK.
Principal investigator email | Email of the PI.
REK number | The REK number under which the data is collected. The REK will also contain the start and end-dates for data collection. Cristin.no for example can be used as a reference.
Project start date | Start date of the projects data collection.
Project end date | End date of the projects data collection.
What is supposed to happen at the end of the project? | Two options are provided: "Delete data" and "Full anonymization and make data available". Whereas the "Delete data" option will trigger a workflow that requests confirmation and a copy of all the data stored at the end of the specified project time the "Full anonymization" option is currently undefined. It is not clear what full anonymization means in the context of every study (de-facing, removal of all DICOM header data, ...).

This information is captured in a web-form and entered after manual review as a record into a REDCap project "DataTransferProjects". See the complete data dictionary for this project:

 - [components/CreateProject/DataTransferProjects.csv](components/CreateProject/DataTransferProjects.csv)

This setup has three instruments. One which is used to capture the text information, correspondence with the PI. One which contains the entries from the form as well as the API link to the projects database and a list of additional *Exclusion* instruments. Those are used to configure series description pattern where the automatic removal of burned in image information is not performed.

Additionally to storing the information as a DataTransferProject a separate REDCap project is created under the projects acronym that follows the event structure requested by the PI of the project. The project setup is a minimal longitudinal setup with the participant ID as the record id and the event names as specified in the project creation request. A *basic demographic form* is used to store a record of all incoming image data. Additional instruments are created based on the project needs. This includes for example measures extracted from secondary captures or structured reports.

### Study data model

A *study* or *DICOM study* refers to data collected from a participant at a point in time. The unique key used to reference a study is the StudyInstanceUID, a code that is created by the imaging device using information from the software, the time of the data collection (unix time stamp) and some random values. As such it is assumed that the StudyInstanceUID is unique in the universe.

The research information system is using the StudyInstanceUID as record id for all incoming image data.

In order to guarantee the separation of data into projects the research information system uses a quarantine system called "FIONA" (Flash based Input/Output Network Applicance) that is the only destination for data into the research PACS.



Item | Description
-----|------------
StudyInstanceUID | Record ID used to identify an incoming image study.
StudyInstitutionName | Institution name in the incoming study. This name is replaced in the imported study (see pseudonymization) with the project acronym.
AETitle of the sending DICOM node | The application entity title used by the sending DICOM node.
AETitle the sending node addresses on FIONA | The application entity title the sender wants to reach. Usually this is "FIONA".
Patient name | The name of the participant (sensitive data). This name is the name in the incoming study, not the name pseudonymized name in the research information system.
Patient ID | The participant id from the incoming image series. Both the name and the ID are sensitive data and are not displayed towards the user.
Study description | The study description text from the DICOM study. Such a name is usually not sensitive and based on the scan protocol, but it can be changed on the scanner.
Study date | The date of the study in DICOM format YYYYMMDD.
Study time | The time of the start of the study.
Accession number of the incoming study | A text string that is generated by the clinical information system. Its unique at the institution level and attached to the study when its ordered. Such a value is suitable for data migration as it identifies a particular dataset without containing sensitive information such as the participant name or birth date.

Each study in the research information systems *Incoming* project has one or more *series* attached to it. The information on the series level are:

Item | Description
-----|------------
Series Instance UID | The unique ID of the series. Similar to the StudyInstanceUID this value is guaranteed to be unique in the univers. All individual images that make up a series will have the same SeriesInstanceUID.
Series Description | A textual description of the image series. Similar to the Study description a series description is defined in the scanning protocol but can be changed by the user during data collection on the scanner.
Series Date | The date of the series capture. This is usually the same as the study date.
Series Time | The time when the first image of the series was captured.
Number of files for this series | This information does not reflect an actuall DICOM tag. Instead it is calculated on the FIONA system based on a time-out detecting the end of a study arriving on FIONA. If there is no more image arriving in a 16seconds interval the FIONA system will assume that all series of the study and all images of the series have been received.
Sequence name of the series | The sequence name is defined in the scanning protocol. It can be used to identify the type of image aquired (structural, functional, diffusion).
List of classified/detected image types for this series | Based on rules defined in [processSingleFile3.py](https://github.com/ABCD-STUDY/FIONASITE/blob/master/server/bin/processSingleFile3.py) each image series is classified using a Tag system. A tag could be just the manufacturer or any other kind of text linked to a combination of DICOM attributes describing the series (field-map, localizer, etc.).

Additionally to the Incomings list of studies and list of series for each study a *Transfer request* form is used to store the information of how to pseudonymize an incoming series. The transfer request form is used to document the users wish to import a study as well as the success of the import. In the workflow of importing data first the study is forwarded from the scanner or the clinical PACS to FIONA. From there a transfer is requested and the data is pseudonymized and forwarded to the research PACS.

Item | Description
-----|------------
The date on which the data transfer was requested | The *Assign* application is used by the researcher to start a data pseudonymization. Data transfers are usually finished for a project in minutes.
Project name to which the data should be transferred | This field is filled out by Assign using a selection from the list of existing (and active) projects.
Anonymize patient name in target project | The participant name selected on the *Assign* page. Both DICOM patient id and patient name are set to this name.
The name of the event assigned to this dataset. | The names of events is predefined for a project (see project setup). Each event name is a event_name in the REDCap project. The *Assign* page lists such event names for the particular study. During pseudonymization by FIONA the event name is written into the DICOM tag "Referring physician" (0008,0090).
Date of the transfer | The date and time when the transfer was started (filled out by FIONA).
Transfer errors (last transfer) | Checkboxes for "Anonymization error", "Send error to rDMA" and "Performed pixel anonymiztation".
StudyInstanceUID in the anonymized DICOM files | The pseudonymized study instance uid written into the DICOM files by FIONA during the forwarding step to the research PACS.
Message generated from last error (anonymize or send error) | A JSON structure containing the pseudonymization and forwarding messages generated by the system processes that process and forward images from the quarantine system FIONA to the research PACS.

The three instruments of the Incoming project are used for all data migration into the research information system. The complete data dictionary can be found here:

 - [components/DataMigration/Incoming_DataDictionary.csv](components/DataMigration/Incoming_DataDictionary.csv)

