# Documentation for medical device development

_Software is considered a medical device. A potential commercialization as patent or medical device requires a minimum on documentation that covers the full life-cycle of the (potential) product. The following is a collection of common tools to structure such a development. This document does not cover all legal documents, specifically CE marking is not explicitly covered._

---

## Overview

Explain a general setup for individual research projects. Not every project needs all of these artefacts. All research projects should be able to point to relevant documentation if asked.

## Key Points

- Device identification including risk classification and intended purpose
- Documented software design, development and quality control

### Risk classification

For EU and international (IMDRF) frameworks the classification of software as a medical device focuses on two cases. Your software is either Class IIa (medium risk) or Class IIb (medium to high risk). If your software for example suggest a procedure, or generates evidence that a human might take for advice, it is considered high risk (Class IIb).

### Intended purpose

The intended purpose of a medical device is a clear single sentence. It is similar to a mission statement but directed towards a single outcome for the software. Here an example: _BLABLA is used in aortic aneurism repair to assess the quality of an implant using a single prognostic numeric value._ 

### Documented software development

Add explicit technical documentation to your software development process. The documentation is supposed to be a living document that reflects the development status for a release. Any software release should include a release of all technical documentation.

Each document should be under versioning and have a version history section. Each entry in the document should include who/when it was added. Use tables to structure individual items and provide a number to identify individual entries. Reference the numbers across different documents, e.g. an issue might development into a new risk entry. A new risk might require a design change. A software update require additional new tests. Make sure all of the identifiers link to each other in this chain of evidence. It is not important what technical tool you use (Word/Pages/Google Docs/Markdown) but, it is suggested to have a _document management system_ that can keep all releases of documents over time.

- **Device description**: A detailed description of any innovative or novel features.
- **Design document**: List all requirements, who generated each. Add use-cases. Describe software architecture. 
- **Risk management**: List all identified risks. Add more as soon as you are discovering them.
- **Issue tracking**: List all feature requests or bug reports, who, what and how to replicate. Add resolution (fixed, could-not-be-replicated, will-not-be-fixed).
- **Version control**: Use software version control (git). Reference issues as (fix #N), risk id, and design ids as comment in your source code. Add commit id to resolved / closed issues.
- **Quality management system**: List all individual tests with setup, individual steps to replicate and expected outcomes. Unexpected outcomes should lead back to new issues.

##### Device description

Include what the product is called and how it works. Describe any innovative or novel features. Describe the patient population, include age, sex, weight, health conditions or any other selection criteria. Add a description on how the device achives its purpose.

##### Design

Add a header to the document that lists the name of the document, the current version, date of last change, author name and signature fields.

Add a design entry table with columns: Number of entry, Description of requirement, Stakeholder. 

##### Risk Management

Add a header to the document that lists the name of the document, the current version, date of last change, author name and signature fields.

Add a risk entry table with columns: Number of entry, Description of risk, Stakeholder, Risk severity, Likelihood of risk.

Use a 5 point scale for risk severity and risk likelihood from 1 (low) to 5 (very high).

##### Issue Tracking

Add a header to the document that lists the name of the document, the current version, date of last change, author name and signature fields.

Add a issue list table with columns: Issue number, Description, Resolution status.

For resolution status use the status tags from github including "Feature", "Bug", "Documentation".

##### Quality Management

Add a header to the document that lists the name of the document, the current version, date of last change, author name and signature fields.

Add a test entry table with columns: Test number, Description on how to replicate, Expected response, Issue description.

For multi-stage tests include a step counter in the description and separate different tests using a merged row with the test name. Reference the issue or risk number that resulted in each quality management entry.

_Last updated: Hauke Bartsch, 2026-05-29_
