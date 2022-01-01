#ifdef __cplusplus
#define EXPORT extern "C"
#else
#define EXPORT extern
#endif

#include <stdbool.h>
#include <stdlib.h>

EXPORT bool isRunning(char* name);
EXPORT void killIfRunning(char *name);
EXPORT void focus(char *name);