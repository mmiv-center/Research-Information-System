import pydicom
import glob
import numpy as np
import sys
import os
import json
import matplotlib.pyplot as plt


description = {}
with open(os.path.join(sys.args[1], "descr.json")) as f:
    description = json.load(f)

files = []
print('glob: {}/input'.format(sys.argv[1]))
for fname in glob.glob(sys.argv[1]+"/input", recursive=False):
    print("loading: {}".format(fname))
    files.append(pydicom.dcmread(fname))

print("file count: {}".format(len(files)))

slices = sorted(files, key=lambda s: s.SliceLocation)

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
plt.imshow(img3d[:, :, img_shape[2]//2])
a1.set_aspect(ax_aspect)

a2 = plt.subplot(2, 2, 2)
plt.imshow(img3d[:, img_shape[1]//2, :])
a2.set_aspect(sag_aspect)

a3 = plt.subplot(2, 2, 3)
plt.imshow(img3d[img_shape[0]//2, :, :].T)
a3.set_aspect(cor_aspect)

plt.show()
input("Press Enter to continue...")

# store any result data in sys.argv[1]/output
output=os.path.join(sys.argv[1],"output")
if not(os.path.exists(output)):
    try:
        os.mkdir(output,0o770)
    except OSError as error:
        print(error)

