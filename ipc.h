#include <sys/stat.h>
#include <sys/types.h>

/*
    Posix
*/

int astikit_close(int fd, int *errno_ptr);
int astikit_fstat(int fd, struct stat *s, int *errno_ptr);
int astikit_ftruncate(int fd, off_t length, int *errno_ptr);
void *astikit_mmap(size_t length, int fd, int *errno_ptr);
int astikit_munmap(void *addr, size_t length, int *errno_ptr);
int astikit_shm_open(char *name, int flags, mode_t mode, int *errno_ptr);
int astikit_shm_unlink(char *name, int *errno_ptr);

/*
    System V
*/

int astikit_ftok(char *path, int project_id, int *errno_ptr);
int astikit_sem_get(key_t key, int flags, int *errno_ptr);
int astikit_sem_close(int id, int *errno_ptr);
int astikit_sem_lock(int id, int *errno_ptr);
int astikit_sem_unlock(int id, int *errno_ptr);
void *astikit_shm_at(int id, int *errno_ptr);
int astikit_shm_get(key_t key, int size, int flags, int *errno_ptr);
int astikit_shm_close(int id, const void *addr, int *errno_ptr);