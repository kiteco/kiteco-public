#ifdef __cplusplus
#define EXPORT extern "C"
#else
#define EXPORT extern
#endif

#include <stdbool.h>
#include <stdlib.h>

EXPORT bool windowVisible(char* name);