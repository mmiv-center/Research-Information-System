#!/usr/bin/env python

import pydicom as dicom
from datetime import datetime
import sys, os, json

if len(sys.argv) != 2:
    print("Usage: <path to data>")
    sys.exit(0)

path=sys.argv[1]
script_directory = os.path.dirname(os.path.abspath(sys.argv[0]))

existingStudies = {}
for root, _, filenames in os.walk(path):
    for filename in filenames:
        try:
            #print("try to read %s" % (root+"/"+filename))
            ds = dicom.dcmread(root+"/"+filename, force=True)
        except IOError as e:
            #print("Error reading this file %s" % (filename))
            continue
        try:
            PatientID=ds['PatientID'].value
            StudyInstanceUID=ds['StudyInstanceUID'].value
            SeriesInstanceUID=ds['SeriesInstanceUID'].value
        except KeyError as e:
            continue
        else:
            key=PatientID+StudyInstanceUID+SeriesInstanceUID
            existingStudies[key] = { "PatientID": PatientID, "StudyInstanceUID": StudyInstanceUID, "SeriesInstanceUID": SeriesInstanceUID }

with open("%s/study_cache.json" % (script_directory), "w") as outfile:
    json_object = json.dumps(list(existingStudies.values()), indent=2)
    outfile.write(json_object)

