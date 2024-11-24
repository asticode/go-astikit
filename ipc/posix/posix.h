#include <sys/stat.h>
#include <sys/types.h>

int astikit_close(int fd, int *errno_ptr);
int astikit_fstat(int fd, struct stat *s, int *errno_ptr);
int astikit_ftruncate(int fd, off_t length, int *errno_ptr);
void *astikit_mmap(size_t length, int fd, int *errno_ptr);
int astikit_munmap(void *addr, size_t length, int *errno_ptr);
int astikit_shm_open(char *name, int flags, mode_t mode, int *errno_ptr);
int astikit_shm_unlink(char *name, int *errno_ptr);