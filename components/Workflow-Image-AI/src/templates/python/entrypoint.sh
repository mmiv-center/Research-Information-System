#!/bin/bash --login
# The --login ensures the bash configuration is loaded.

# Expect one argument (the name of the conda environment)
if [ ! -z "$CONDA_DEFAULT_ENV" ]; then
  conda_env="$CONDA_DEFAULT_ENV"
fi

if [ -z "$conda_env" ]; then
    echo "Usage: <conda-env>"
    exit -1
fi

# where is pr2mask?
export PATH="/pr2mask:$PATH"

# if we find imageAndMask2Report and json2SR in this container
auto_report_mode=0
output="/output"
output2="/tmp/output"
if [ -f /pr2mask/imageAndMask2Report ]; then
    # enable the automatic report generation
    auto_report_mode=1
    if [ ! -d "${output2}" ]; then
        mkdir "${output2}"
    fi
else
    output2="${output}"
fi

# Temporarily disable strict mode and activate conda:
set +euo pipefail
conda activate "${conda_env}"
if [ $? -ne 0 ]; then
   echo "Error: activating conda environment \"$conda_env\" failed."
   exit -1
fi
set -euo pipefail

# execute the python command, provide as argument the 
#   directory that is mounted on /data
#   directory that is mounted on /output
log_file="${output}"/stub_command.log
cmd="$@ ${output2}"
echo "run now: $cmd"
eval $cmd

if [ "$auto_report_mode" -eq 1 ]; then
    # only if we have access to pr2mask features we can do the following
    /pr2mask/imageAndMask2Report /data/input "${output2}/mask" "${output2}" -u -i "$VERSION" --reporttype mosaic >> "${log_file}" 2>&1
    /pr2mask/json2SR "${output2}"/*.json >> "${log_file}" 2>&1
    cp -R "${output2}"/fused "${output}"
    cp -R "${output2}"/labels "${output}"
    cp -R "${output2}"/reports "${output}"
    cp -R "${output2}"/redcap "${output}"
    cp -R "${output2}"/*.dcm "${output}"
    chmod -R 777 /output
fi
echo "$(date): processing done" >> "${log_file}"
