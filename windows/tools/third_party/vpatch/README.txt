NSIS Patch Generation Utility v1.0.2
====================================

The nsisPatchGen utility recursively compares two directory structures
looking for changes to the files and subdirectories. It produces an 
NSIS include file containing functions that will perform a patch 
upgrade from the original structure to the new.

NsisPatchGen uses the VPatch "genpat.exe" utility to generate patch files.
These files contain the delta between the old and new version of an
individual file. When applied to the old file, the file differences are 
applied and the file is converted to its new contents.


Version History
---------------
1.0.0   09/05/2006	Initial version.
1.0.1	02/09/2007	Bug fixes.
1.0.2	10/11/2007	Bug fixes.

See CHANGES.txt	for details.


Usage:
------

nsisPatchGen [--hidden] [--system] directory1 directory2 [patch output directory] [NSIS output file]

--hidden              : include hidden files in comparison
--system              : include system files in comparison
directory1            : the top level directory containing the original set of files
directory2            : top level directory containing the target set of files
patch output direcory : directory into which the patch files will be placed (default "patchFiles")
NSIS output file      : name of the generated NSIS file (default "patchFiles.nsi")

Notes:
------

+ nsisPatchGen will treat changes in file or directory name case as the file being removed and a new one added.
In the future we may fully support case changes.


Pre-requisites
--------------

1. Install NSIS and the VPatch plug-in.
2. Define the environment variable VPATCH containing a reference to the directory
containing genpat.exe
3. Add "%VPATCH" to the PATH environment variable.

Using nsisPatchGen
------------------

nsisPatchGen recursively diffs two sets of directories to calculate the delta from
one to the other, and to produce a patch installer that will upgrade from one to 
the other.

- "directory1" is the top level directory containing the old version of the 
software. This should contain the files in the state they would be installed on 
a target machine. 

- "directory2" is the equivalent top level directory containing the update version
of the same software. 

Run: 

  nsisPatchGen <directory1> <directory2>

This will produce an NSIS include file, "patchFiles.nsi", and a number of patch file
in a "patchFiles" directory.

This include contains a number of NSIS functions:

  patchDirectoriesAdded     : creates all that exist in directory2 but not in directory1
  patchDirectoriesRemoved   : removes any subdirectories that do not exist in directory2
  patchFilesAdded           : installs files that exist in directory2 but not in directory1
  patchFilesRemoved         : removes files that exist in directory1 but not in directory2
  patchFilesModified        : patches any files that have been modified.
  
It is recommended that the patch functions are called in the following order:

  patchFilesRemoved
  patchDirectoriesRemoved
  patchDirectoriesAdded
  patchFilesAdded
  patchFilesModified
  


The calling script must define the following variables _before_ calling any
of the patch functions:

The location of <directory2> on the source machine, e.g.
  
  !define PATCH_SOURCE_ROOT "newVersion"
  
 The location of the *.pat patch files on the source machine (as defined by the 
 optional <patch output directory> command line parameter), e.g.
 
  !define PATCH_FILES_ROOT "patchFiles"
  
 The directory to which the files will be installed, e.g.

  !define PATCH_INSTALL_ROOT $INSTDIR

The example file "testInstaller.nsi" contains an example NSIS installer script
that uses a patchFile.nsi include file to perform a patch install.


Building nsisPatchGen
---------------------

Requirements:	Qt 4.1.4
Optional:		Microsoft Visual Studio .NET 2003 / 2005 Express

				The unit tests require:
				cppunit 1.11.6 (http://sourceforge.net/projects/cppunit)
				qxrunner 0.2.2 (http://qxrunner.systest.ch)
				The above should be built and you need to declare CPPUNITDIR and QXRUNNERDIR
				enviroment variables pointing to the root of cppunit / qxrunner.

Building Using Visual Studio
----------------------------

Open the solution file "nsisPatchGen_vs2003.sln" or "nsisPatchGen_vs2005E.sln" depending on your version of visual studio. 

Select "Build-->Build Solution".

This will build nsisPatchGen and a set of unit tests.


Building Using QMake
--------------------

From the consle, cd to the src directory. Then run "qmake". This will produce the
makefile. Now run your make tool (eg "nmake" or "nmake debug" for a debug build).

This will build nsisPatchGen.exe.

To build and run the unit tests run "qmake CONFIG+=test" from the src directory.
Then run your make tool as described above. 

This will build nsisPatchGen-test.exe
The tests expect to run with the working directory src\test so:

cd test
..\debug\nsisPatchGen-test.exe


LICENSE
=======

nsisPatchGen
Copyright (C) 2006  Vibration Technology Ltd. ( http://www.vibtech.co.uk )

This program is free software; you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation; either version 2 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program; if not, write to the Free Software
Foundation, Inc., 51 Franklin Street, Fifth Floor, Boston, MA  02110-1301  USA


