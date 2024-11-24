#include <sys/types.h>

int astikit_ftok(char *path, int project_id, int *errno_ptr);
int astikit_sem_get(key_t key, int flags, int *errno_ptr);
int astikit_sem_close(int id, int *errno_ptr);
int astikit_sem_lock(int id, int *errno_ptr);
int astikit_sem_unlock(int id, int *errno_ptr);
void *astikit_shm_at(int id, int *errno_ptr);
int astikit_shm_get(key_t key, int size, int flags, int *errno_ptr);
int astikit_shm_close(int id, const void *addr, int *errno_ptr);