#include <errno.h>
#include <stdlib.h>
#include <string.h>
#include <sys/mman.h>
#include <sys/sem.h>
#include <sys/shm.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <unistd.h>

/*
    Posix
*/

int astikit_close(int fd, int *errno_ptr)
{
    int ret = close(fd);
    if (ret < 0)
    {
        *errno_ptr = errno;
    }
    return ret;
}

int astikit_fstat(int fd, struct stat *s, int *errno_ptr)
{
    int ret = fstat(fd, s);
    if (ret < 0)
    {
        *errno_ptr = errno;
    }
    return ret;
}

int astikit_ftruncate(int fd, off_t length, int *errno_ptr)
{
    int ret = ftruncate(fd, length);
    if (ret < 0)
    {
        *errno_ptr = errno;
    }
    return ret;
}

void *astikit_mmap(size_t length, int fd, int *errno_ptr)
{
    void *addr = mmap(NULL, length, PROT_READ | PROT_WRITE, MAP_SHARED, fd, 0);
    if (addr == MAP_FAILED)
    {
        *errno_ptr = errno;
        return NULL;
    }
    return addr;
}

int astikit_munmap(void *addr, size_t length, int *errno_ptr)
{
    int ret = munmap(addr, length);
    if (ret < 0)
    {
        *errno_ptr = errno;
    }
    return ret;
}

int astikit_shm_open(char *name, int flags, mode_t mode, int *errno_ptr)
{
    int fd = shm_open(name, flags, mode);
    if (fd < 0)
    {
        *errno_ptr = errno;
    }
    return fd;
}

int astikit_shm_unlink(char *name, int *errno_ptr)
{
    int ret = shm_unlink(name);
    if (ret < 0)
    {
        *errno_ptr = errno;
    }
    return ret;
}

/*
    System V
*/

int astikit_ftok(char *path, int project_id, int *errno_ptr)
{
    int key = ftok(path, project_id);
    if (key < 0)
    {
        *errno_ptr = errno;
    }
    return key;
}

int astikit_sem_get(key_t key, int flags, int *errno_ptr)
{
    int id = semget(key, 1, flags);
    if (id < 0)
    {
        *errno_ptr = errno;
    }
    return id;
}

int astikit_sem_close(int id, int *errno_ptr)
{
    int ret = semctl(id, 0, IPC_RMID);
    if (ret < 0)
    {
        *errno_ptr = errno;
    }
    return ret;
}

// "0" means the resource is free
// "1" means the resource is being used

int astikit_sem_lock(int id, int *errno_ptr)
{
    struct sembuf operations[2];

    // Wait for the value to be 0
    operations[0].sem_num = 0;
    operations[0].sem_op = 0;
    operations[0].sem_flg = 0;

    // Increment the value
    operations[1].sem_num = 0;
    operations[1].sem_op = 1;
    operations[1].sem_flg = 0;

    int ret = semop(id, operations, 2);
    if (ret < 0)
    {
        *errno_ptr = errno;
    }
    return ret;
}

int astikit_sem_unlock(int id, int *errno_ptr)
{
    struct sembuf operations[1];

    // Decrement the value
    operations[0].sem_num = 0;
    operations[0].sem_op = -1;
    operations[0].sem_flg = 0;

    int ret = semop(id, operations, 1);
    if (ret < 0)
    {
        *errno_ptr = errno;
    }
    return ret;
}

int astikit_shm_get(key_t key, int size, int flags, int *errno_ptr)
{
    int id = shmget(key, size, flags);
    if (id < 0)
    {
        *errno_ptr = errno;
    }
    return id;
}

void *astikit_shm_at(int id, int *errno_ptr)
{
    void *addr = shmat(id, NULL, 0);
    if (addr == (void *)-1)
    {
        *errno_ptr = errno;
        return NULL;
    }
    return addr;
}

int astikit_shm_close(int id, const void *addr, int *errno_ptr)
{
    int ret;
    if (addr != NULL)
    {
        ret = shmdt(addr);
        if (ret < 0)
        {
            *errno_ptr = errno;
            return ret;
        }
    }
    ret = shmctl(id, IPC_RMID, NULL);
    if (ret < 0)
    {
        *errno_ptr = errno;
    }
    return ret;
}