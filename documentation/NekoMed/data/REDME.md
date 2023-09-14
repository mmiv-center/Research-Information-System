# Example dataset

The dataset is not part of this repository but can be obtained as the SIIM hackathon-dataset, see the repository mentioned below. 

```bash
cd NekoMed
mkdir data; cd data
git clone https://github.com/ImagingInformatics/hackathon-dataset.git
cd hackathon-dataset
git submodule update --init --recursive
```

Any other repository of DICOM data would work as well. We rely on different image studies being stored in different folders.