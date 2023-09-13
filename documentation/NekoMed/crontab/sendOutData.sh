#!/usr/bin/env bash

# pick a connection and pick a dataset (folder)
# send out randomly

# check if the connection is active
cons="../php/connections.json"
data="../data/"

if [ ! -e "${cons}" ]; then
    echo "Info: no \"${cons}\" found, nothing to be done."
    exit 0
fi

numConnections=$(jq length "$cons")
if [ "${numConnections}" = "" ]; then
    echo "Info: no connection yet, nothing to be found"
    exit 0
fi
if [ "${numConnections}" -eq 0 ]; then
    echo "Info: no connection yet, nothing to be found"
    exit 0
fi

pickedConnection=$(($RANDOM % numConnections))
echo "There are $numConnections connections available. We select connection $pickedConnection."
IP=$(jq ".[$pickedConnection].IP" "$cons" | tr '"' ' ')
Port=$(jq ".[$pickedConnection].Port" "$cons" | tr '"' ' ')
AETitle=$(jq ".[$pickedConnection].AETitle" "$cons")
numSeries=$(jq ".[$pickedConnection].SeriesReceived" "$cons" | tr '"' ' ')

# We should check if that connection is active before we send something.
echo "Try to reach $IP $AETitle $Port"
/opt/homebrew/bin/echoscu -q -to 1 --call $AETitle $IP $Port &> /dev/null
if [ $? -eq 0 ]; then
    echo "working, send some data"
    tmp_file=$(mktemp)
    jq --argjson pickedConnection "${pickedConnection}" --arg working 1 '.[$pickedConnection].SCUWorking = $working' "$cons" > "${tmp_file}"
    # move the file into the correct position
    if [ -s "${tmp_file}" ]; then
	mv "${tmp_file}" "$cons"
    fi
else
    echo "Connection could not be established, no DICOM listener found."
    tmp_file=$(mktemp)
    jq --argjson pickedConnection "${pickedConnection}" --arg working 0 '.[$pickedConnection].SCUWorking = $working' "$cons" > "${tmp_file}"
    # move the file into the correct position
    if [ -s "${tmp_file}" ]; then
	mv "${tmp_file}" "$cons"
    fi
    exit -1
fi

# pick a dataset by random
datasets=$(find "$data" -type f -name "*.dcm" -exec dirname "{}" \; | sort -u | sed -e 's/$/\\n/')
numData=$(echo -e $datasets | wc -l | tr -d ' ')
pickedData=$(($RANDOM % numData))
path=$(echo -e $datasets | sed "${pickedData}q;d" | sed 's/^[ \t]*//')
echo "Picked path [$pickedData/$numData] is: \"$path\""

/opt/homebrew/bin/storescu -nh +r +sd --timeout 2 -aec "$AETitle" -aet "NekoMed" $IP $Port "$path" &> /dev/null
if [ $? -eq 0 ]; then
    echo "Worked, done"
else
    echo "Could not send using storescu"
    echo "Command line was:"
    echo "/opt/homebrew/bin/storescu -nh +r +sd --timeout 2 -aec \"$AETitle\" -aet \"NekoMed\" $IP $Port \"$path\""
    exit -1
fi

# We should update to get some stats done...
tmp_file=$(mktemp)
newNumSeries=$((numSeries+1))
jq --argjson pickedConnection "${pickedConnection}" --arg newNumSeries "$newNumSeries" '.[$pickedConnection].SeriesReceived = $newNumSeries' "$cons" > "${tmp_file}"
# move the file into the correct position
if [ -s "${tmp_file}" ]; then
    mv "${tmp_file}" "$cons"
fi
