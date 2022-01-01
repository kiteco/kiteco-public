// +build !windows

package readdir

/*
#include <stdlib.h>
#include <stdio.h>
#include <stdbool.h>
#include <string.h>

#include <sys/types.h>
#include <sys/stat.h>
#include <unistd.h>
#include <dirent.h>

struct simple_dirent {
	char name[1024];
	bool dir;
	bool d_type_known;
};

struct simple_dirent* simple_readdir(const char* filename, int* entries) {
	DIR *dir;
	struct dirent *dp;

	dir = opendir(filename);
	if (dir == NULL) {
		return NULL;
	}

	int count = 0;
	while ((dp = readdir(dir)) != NULL) {
		if (strcmp(dp->d_name, ".") == 0 || strcmp(dp->d_name, "..") == 0) {
			continue;
		}
		count++;
	}

	rewinddir(dir);

	int i = 0;
	struct simple_dirent *vals = malloc(sizeof(struct simple_dirent)*count);
	while ((dp = readdir(dir)) != NULL && i < count) {
		if (strcmp(dp->d_name, ".") == 0 || strcmp(dp->d_name, "..") == 0) {
			continue;
		}

		strcpy(vals[i].name, dp->d_name);
		vals[i].dir = (dp->d_type == DT_DIR);
		vals[i].d_type_known = (dp->d_type != DT_UNKNOWN);
		i++;
	}

	closedir(dir);

	*entries = count;
	return vals;
}
*/
import "C"
import "unsafe"

// List returns entries of the given directory
func List(path string) []Dirent {
	dir := C.CString(path)
	defer C.free(unsafe.Pointer(dir))

	var count C.int
	ents := C.simple_readdir(dir, &count)
	defer C.free(unsafe.Pointer(ents))

	if count == 0 {
		return nil
	}

	/*
		    From: SO link: /questions/28925179/cgo-how-to-pass-struct-array-from-c-to-go

				The easiest way to use a C array in go is to convert it to a slice through an array:

					team := C.get_team()
					C.free(unsafe.Pointer(team))
					teamSlice := (*[1 << 30]C.team)(unsafe.Pointer(team))[:teamSize:teamSize]

				The max-sized array isn't actually allocated, but Go requires constant size arrays,
				and 1<<30 is going to be large enough. That array is immediately converted to a slice,
				with the length and capacity properly set.
	*/

	slice := (*[1 << 30]C.struct_simple_dirent)(unsafe.Pointer(ents))[:count:count]

	var ret []Dirent
	for _, s := range slice {
		ret = append(ret, Dirent{
			Path:         C.GoString(&s.name[0]),
			IsDir:        bool(s.dir),
			DTypeEnabled: bool(s.d_type_known),
		})
	}

	return ret
}
