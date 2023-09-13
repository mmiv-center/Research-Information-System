# NekoMed - MMIV Connectathon for image-based AI systems

## Setup

Serve the local index.php from a php server on a port such as 4444. You will need to replace "localhost" with the IP address of the hosting machine once you are connected to the competition network.

```bash
cd NekoMed
php -S localhost:4444
```

```bash
cd php
watch -n 2 ./sendOutData.sh
```

## Client

Connect to the network and Navigate to the website:

```
http://localhost:4444/
```

```bash
cd php
storescp -aet "DOG" \
         --sort-on-study-uid "MYDATA" \
         --eostudy-timeout 16 \
         --exec-sync \
         --exec-on-eostudy "/Users/haukebartsch/src/Research-Information-System/documentation/NekoMed/php/process.py #r #a #c #p" \
         -od "/tmp/" \
         -fe ".dcm" \
         11114
```