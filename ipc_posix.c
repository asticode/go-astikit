#include <errno.h>
#include <sys/mman.h>
#include <sys/stat.h>
#include <unistd.h>

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