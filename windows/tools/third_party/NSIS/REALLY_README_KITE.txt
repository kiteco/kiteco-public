a bunch of things are non-obvious about NSIS releases, so be careful.

* this is the Unicode fork of NSIS.
* it is not the 64-bit fork.


* this build also has a larger maximum string length.
* there are a couple of additions to Include and Plugins, including FindProcDLL, KillProcDLL (both built for Unicode NSIS), GetProcessInfo, and servicelib.


* I customized GetProcessInfo and servicelib for the Unicode build (mostly making sure System:: calls are passing strings appropriately and not calling the ANSI variants).
* some files are missing, e.g. in Contrib\Language files, in order to minimize the number of files in the repo.  They shouldn't be dangerous to add as needed.