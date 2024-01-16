#!/bin/bash --login
# The --login ensures the bash configuration is loaded.

# Expect one argument (the name of the conda environment)
conda-env=""
if [ "$#" -eq 1 ]; then
    conda-env="$1"
else
    echo "Usage: <conda-env>"
    exit -1
fi

# where is pr2mask?
export PATH="/pr2mask:$PATH"

# if we find imageAndMask2Report and json2SR in this container
auto_report_mode=0
output2="${output}"
if [ command -v imageAndMask2Report ] && [ command -v json2SR ]; then
    # enable the automatic report generation
    auto_report_mode=1
    if [ ! -d "${output2}" ]; then
        mkdir "${output2}"
    fi
fi

# Temporarily disable strict mode and activate conda:
set +euo pipefail
conda activate $1
if [ $? -ne 0 ]; then
   echo "Error: activating conda environment \"$1\" failed."
   exit -1
fi
set -euo pipefail

# execute the python command, provide as argument the 
#   directory that is mounted on /data
#   directory that is mounted on /output
log_file="${output}"/stub_command.log
exec python stub.py /data "${output2}" >> "${log_file}" 2>&1

if [ "$output_report_mode" -eq 1 ]; then
    # only if we have access to pr2mask features we can do the following
    imageAndMask2Report /data/input "${output2}" -u >> "${log_file}" 2>&1
    json2SR "${output2}"/*.json >> "${log_file}" 2>&1
    cp -R "${output2}"/fused "${output}"
    cp -R "${output2}"/labels "${output}"
    cp -R "${output2}"/report "${output}"
    cp -R "${output2}"/redcap/*/output.json "${output}"
    cp -R "${output2}"/*.dcm "${output}"
    chmod -R 777 /output
fi
echo "$(date): processing done" >> "${log_file}"
