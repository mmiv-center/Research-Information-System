# SELECT Statement Grammar

## Basic Syntax

```
SELECT <output_level> FROM study WHERE SERIES [NAMED <named match>] HAS <conditions> [ALSO WHERE...]...
```

### Output Levels
- `patient` - Return all studies for the matching patients
- `study` - Return all series for the matching studies  
- `series` - Return matching series (default)

## Operators

### Comparison Operators
- `==` - Exact match, if value is a list any element may match
- `regexp` - Regular expression match (case-sensitive)
- `containing` - Exact match, if value is a list any element may match
- `>`, `<`, `>=`, `<=` - Numeric comparison

### Logical Operators
- `AND` - All conditions must be true
- `OR` - Any condition can be true
- `NOT` - Negate a condition

## DICOM Tags

### Common Queryable Tags
- `Modality` - Imaging modality (MR, CT, US, etc.)
- `SeriesDescription` - Series name
- `StudyDescription` - Study name
- `PatientName` - Patient name
- `PatientID` - Patient identifier
- `StudyDate` - Date of study
- `NumImages` - Number of images in series
- `ClassifyType` - Custom classification tag

## Examples by Use Case

### Example 1: Find all MR imaging

```
SELECT series FROM study WHERE SERIES NAMED "MR" HAS Modality == 'MR'
```

### Example 2: Find T1 and T2 weighted images

````
SELECT series FROM study WHERE series named "T1" has 
    Modality = 'MR' AND SeriesDescription regexp 'T1'
ALSO WHERE series named "T2" has 
    Modality == 'MR' AND SeriesDescription regexp 'T2'
````

### Example 3: Complex multi-series filter

```
SELECT patient FROM study
WHERE series named "Anat" has Modality == 'MR' AND SeriesDescription regexp '^Anat'
ALSO WHERE series named "Func" has Modality == 'MR' AND NumImages > 100
ALSO WHERE series named "DTI" has ClassifyType containing DIFFUSION
````

### Example 4: Numeric filtering

```
SELECT series FROM study
WHERE Modality == 'MR'
AND NumImages > 50
AND NumImages < 500
```
