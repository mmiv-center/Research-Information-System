#!/bin/env bash

# We assume we have access to some DICOM tools like dcmtk
# and tools like jq to get and put values into the output.json.

# check if the input path exists
if [ ! -d "${1}" ]; then
    echo "Error: the  input directory \"${1}\" does not exist."
    exit -1
fi

input="${1}/input"
output="${1}/output"

# extract some information from the input DICOM folder


###############################################
# Put your code here. 
#  Data: ${input}
#  Structured Information: ${input}/descr.json
###############################################


