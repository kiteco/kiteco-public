# General
This pipeline dedupes the input stream of python files by their content.

# Input
Key: name of a python file
Value: contents of a python file

# Output
Key: hash of contents of of a python file
Value: contents of a python file

# Notes
- This is a lossy transformation since we lose the name of the source file and the frequency that it was duplicated.