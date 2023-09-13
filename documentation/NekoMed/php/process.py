#!/usr/bin/env python

import pydicom as dicom
from datetime import datetime
import sys, os

callingIP="%s" % (sys.argv[1])
callingAETitle=sys.argv[2]
calledAETitle=sys.argv[3]
path=sys.argv[4]
script_directory = os.path.dirname(os.path.abspath(sys.argv[0]))
#PatientID=""
#StudyInstanceUID=""
#SeriesInstanceUID=""

for root, _, filenames in os.walk(path):
    for filename in filenames:
        try:
            ds = dicom.dcmread(root+"/"+filename, force=True)
        except IOError as e:
            print("Error reading this file %s" % (filename))
            continue
        PatientID=ds['PatientID'].value
        StudyInstanceUID=ds['StudyInstanceUID'].value
        SeriesInstanceUID=ds['SeriesInstanceUID'].value
        break

with open("%s/incoming_data.log" % (script_directory), "a") as log:
    log.write("%s: callingIP=%s, callingAETitle=%s, calledAETitle=%s PatientID=\"%s\" StudyInstanceUID=%s SeriesInstanceUID=%s\n" % (str(datetime.now()), callingIP, callingAETitle, calledAETitle, PatientID, StudyInstanceUID, SeriesInstanceUID))

