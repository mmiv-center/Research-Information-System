# Run the cron-job for sending data and update team entries

We use the command line tool "watch" (brew install watch) to run the script "sendOutData.sh" every 2 seconds.

```bash
watch -n 10 ./sendOutData.sh
```

The script will first select a connection from a team that has been registered. It will test if the connection is active - if the destination responds as a DICOM destination. In the next step the script picks a random dataset (see createStudyCache.py) and sends that study to the destination.

This step takes some time, just like in a hospital. Teams need to keep the connection active for a while to be randomly picked and to receive the data.

### Get competition data and run createStudyCache.py

This script should be run once on the dataset used in the competition. In our competition we used data available in the repository.

```bash
cd NekoMed
mkdir data; cd data
git clone https://github.com/ImagingInformatics/hackathon-dataset.git
cd hackathon-dataset
git submodule update --init --recursive
```

