#!/usr/bin/env bash

# We assume we have access to some DICOM tools like dcmtk
# and tools like jq to get and put values into the output.json.

# check if the input path exists
if [ ! -d "${1}" ]; then
    echo "Error: the  input directory \"${1}\" does not exist."
    exit -1
fi

input="${1}/input"
output="${1}/output"

# check if we have dcmdk and jq
if ! command -v dcmdump &> /dev/null; then
    echo "Error: no dcmtk installed or executables cannot be found in default path"
    exit -1
fi
if ! command -v jq &> /dev/null; then
    echo "Error: no jq installed or executable cannot be found in default path"
    exit -1
fi


# extract some information from the input DICOM folder
file=`ls "${input}"/* | head -1`
ROWS=`dcmdump +P Rows ${file} | cut -d ' ' -f3`
COLS=`dcmdump +P Columns ${file} | cut -d ' ' -f3`

# create output folder
if [ ! -d "${output}" ]; then
    mkdir "${output}"
fi

# make a copy of the description as our starting  output.json
cp "${input}/../descr.json" "${output}"/output.json

# add the matrix information to the output.json
cat "${output}"/output.json | jq --arg shape_x ${ROWS} '. + {shape_x: $shape_x}' > "${output}"/bla.json
mv "${output}"/bla.json "${output}"/output.json

cat "${output}"/output.json | jq --arg shape_y ${COLS} '. + {shape_y: $shape_y}' > "${output}"/bla.json
mv "${output}"/bla.json "${output}"/output.json

# Lets compute an nii file for each of the Series
find "${input}"/../input_view_dicom_series/ -mindepth 2 -maxdepth 2 -type d | xargs -I'{}' dcm2niix {}

##################################################
# Put your code here. 
#  Data: ${input}
#  Structured Information: ${output}/output.json
#  Output DICOM: ${output}/
##################################################

