import pydicom
import glob
import numpy as np
import sys
import os
import json
import matplotlib.pyplot as plt


description = {}
with open(os.path.join(sys.argv[1], "descr.json")) as f:
    description = json.load(f)

files = []
print('glob: {}/input'.format(sys.argv[1]))
for fname in glob.glob(sys.argv[1]+"/input/*", recursive=False):
    #print("loading: {}".format(fname))
    files.append(pydicom.dcmread(fname))

print("file count: {}".format(len(files)))

# make sure we keep data that has the same shape as the first slice
files = [a for a in files if a.pixel_array.shape == files[0].pixel_array.shape]

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
ps = slices[0].PixelSpacing
ss = slices[0].SliceThickness
ax_aspect = ps[1]/ps[0]
sag_aspect = ps[1]/ss
cor_aspect = ss/ps[0]

# create 3D array
img_shape = list(slices[0].pixel_array.shape)
img_shape.append(len(slices))
img3d = np.zeros(img_shape)

# fill 3D array with the images from the files
for i, s in enumerate(slices):
    img2d = s.pixel_array
    img3d[:, :, i] = img2d

# plot 3 orthogonal slices
a1 = plt.subplot(2, 2, 1)
plt.imshow(img3d[:, :, img_shape[2]//2], cmap='gray')
a1.set_aspect(ax_aspect)

a2 = plt.subplot(2, 2, 2)
plt.imshow(img3d[:, img_shape[1]//2, :], cmap='gray')
a2.set_aspect(sag_aspect)

a3 = plt.subplot(2, 2, 3)
plt.imshow(img3d[img_shape[0]//2, :, :].T, cmap='gray')
a3.set_aspect(cor_aspect)

plt.show()

# store any result DICOM data in sys.argv[1]/output
output=os.path.join(sys.argv[1],"output")
if not(os.path.exists(output)):
    try:
        os.mkdir(output,0o770)
    except OSError as error:
        print(error)

#################################################
# This might be a good place to start your work.
# Volume: img3d
# Structured Information: description
#################################################




# Add any structured information to an output.json
# that contains the same information as the descr.json.
description['shape_x'] = img3d.shape[0]
description['shape_y'] = img3d.shape[1]
description['shape_z'] = img3d.shape[2]

# save the structured information into the output folder
with open(output+"/output.json", 'w') as outfile:
    json.dump(description, outfile)
