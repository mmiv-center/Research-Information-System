import pydicom
import glob
import numpy as np
import sys
import os
import json
import matplotlib.pyplot as plt
import matplotlib.gridspec as gridspec

if len(sys.argv) != 3:
    print("Error: wrong number of arguments (%d), should be 2 (input and output folder)." % (len(sys.argv)-1))
    sys.exit(-1)

datafolder = sys.argv[1]
outputfolder = sys.argv[2]

description = {}
descr_path = os.path.join(datafolder, "descr.json")
if os.path.exists(descr_path): 
    with open(descr_path) as f:
        description = json.load(f)[0]

if not(os.path.exists(outputfolder)):
    try:
        print("Info: create output folder \"%s\"" % (outputfolder))
        os.mkdir(outputfolder,0o777)
    except OSError as error:
        (error)
    
files = []
print('glob: {}/input'.format(datafolder))
for fname in glob.glob(datafolder+"/input/*", recursive=False):
    #print("loading: {}".format(fname))
    if os.path.isfile(fname):
        files.append(pydicom.dcmread(fname))

# make sure we only keep data that has the same shape as the first slice
files = [a for a in files if a.get("PixelData") != None and a.pixel_array.shape == files[0].pixel_array.shape]

print("file count: {}".format(len(files)))

# make sure we sort the slices by SliceLocation or, if that does not exist by InstanceNumber
def sortFunc(s):
    if "SliceLocation" in s:
        return s.SliceLocation
    else:
        if "InstanceNumber" in s:
            return s.InstanceNumber
        return 0
slices = sorted(files, key=sortFunc)

# pixel aspects, assuming all slices are the same
ps = slices[0].get("PixelSpacing", [1,1])
ss = slices[0].get("SliceThickness",1)
ax_aspect = ps[1]/ps[0]
sag_aspect = ps[1]/ss
cor_aspect = ps[0]/ss

# create 3D array
img_shape = list(slices[0].pixel_array.shape)
img_shape.append(len(slices))
img3d = np.zeros(img_shape)

# fill 3D array with the images from the files
for i, s in enumerate(slices):
    img2d = s.pixel_array
    img3d[:, :, i] = img2d

# plot 3 orthogonal slices
fig=plt.figure(figsize=(6,6))
fig.patch.set_facecolor('gray')
gs1 = gridspec.GridSpec(2,2)
gs1.update(wspace=0.025, hspace=0.05)

a1 = plt.subplot(gs1[0])
plt.imshow(img3d[:, :, img_shape[2]//2], cmap='gray')
a1.set_aspect(ax_aspect)
a1.set_xticklabels([])

a2 = plt.subplot(gs1[1])
plt.imshow(img3d[:, img_shape[1]//2, :], cmap='gray')
a2.set_aspect(sag_aspect)
a2.set_xticklabels([])
a2.set_yticklabels([])

a3 = plt.subplot(gs1[2])
plt.imshow(img3d[img_shape[0]//2, :, :], cmap='gray')
a3.set_aspect(cor_aspect)
a3.set_yticklabels([])

plt.show()

####################################################
# This might be a good place to add your work.
# Volume: img3d
# Structured Information: description
# As an example here we compute the signal-to-noise
# over the whole volume.
####################################################

sd = img3d.std()
# To import values into the database we need to specify where the data should be stored.
description['signal-to-noise'] = {
    'record_id':  description['PatientID'],
    'redcap_event_name': description['ReferringPhysician'],
    'field_name': 'signal-to-noise',
    'value':      np.where(sd == 0, 0, img3d.mean()/sd).item(),
}
# Some other values that we don't want to import into the database
description['shape_x'] = img3d.shape[0]
description['shape_y'] = img3d.shape[1]
description['shape_z'] = img3d.shape[2]

# remember to save the structured information into the output folder
with open(outputfolder+"/output.json", 'w') as outfile:
    outfile.write(json.dumps(description, indent=4, sort_keys=True))
