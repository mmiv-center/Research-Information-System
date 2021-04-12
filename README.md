# A Research Information System

This repository contains components developed for a research information system using clinical technology.

### Project description

Our research information system is modeled after information systems used in clinical practice for industrial scale data processing, e.g. for the capture, storage and analysis of data for hundreds of thousands of participants per month and millions of participants per year. We adapted the system for research organizations that serve hundreds of research projects.

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
What is supposed to happen at the end of the project? | Supported are two options: "Delete data" and "Full anonymization and make data available"

This information is captured in a web-form and entered after manual review into a REDCap project "DataTransferProjects". See the complete data dictionary for this project at

 - [components/CreateProject/DataTransferProjects.csv](components/CreateProject/DataTransferProjects.csv)
