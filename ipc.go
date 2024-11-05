//go:build !windows

package astikit

//#include <errno.h>
//#include <stdlib.h>
//#include <string.h>
//#include <sys/sem.h>
//#include <sys/shm.h>
//#include <sys/types.h>
/*

int astikit_ftok(char* path, int project_id, int* errno_ptr) {
	int key = ftok(path, project_id);
	if (key < 0) {
		*errno_ptr = errno;
	}
	return key;
}

int astikit_sem_get(int key, int flags, int* errno_ptr) {
	int id = semget(key, 1, flags);
	if (id < 0) {
		*errno_ptr = errno;
	}
	return id;
}

int astikit_sem_close(int id, int* errno_ptr) {
	int ret = semctl(id, 0, IPC_RMID);
	if (ret < 0) {
		*errno_ptr = errno;
	}
	return ret;
}

// "0" means the resource is free
// "1" means the resource is being used

int astikit_sem_lock(int id, int* errno_ptr) {
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
	if (ret < 0) {
		*errno_ptr = errno;
	}
	return ret;
}

int astikit_sem_unlock(int id, int* errno_ptr) {
	struct sembuf operations[1];

	// Decrement the value
	operations[0].sem_num = 0;
	operations[0].sem_op = -1;
    operations[0].sem_flg = 0;

	int ret = semop(id, operations, 1);
	if (ret < 0) {
		*errno_ptr = errno;
	}
	return ret;
}

int astikit_shm_get(int key, int size, int flags, int* errno_ptr) {
	int id = shmget(key, size, flags);
	if (id < 0) {
		*errno_ptr = errno;
	}
	return id;
}

void* astikit_shm_at(int id, int* errno_ptr) {
	void* addr = shmat(id, NULL, 0);
	if (addr == (void*) -1) {
		*errno_ptr = errno;
		return NULL;
	}
	return addr;
}

int astikit_shm_close(int id, const void* addr, int* errno_ptr) {
	int ret;
	if (addr != NULL) {
		ret = shmdt(addr);
		if (ret < 0) {
			*errno_ptr = errno;
			return ret;
		}
	}
	ret =  shmctl(id, IPC_RMID, NULL);
	if (ret < 0) {
		*errno_ptr = errno;
	}
	return ret;
}

*/
import "C"
import (
	"errors"
	"fmt"
	"os"
	"sync"
	"syscall"
	"unsafe"
)

func NewSystemVIpcKey(projectID int, path string) (int, error) {
	// Get c path
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))

	// Get key
	var errno C.int
	key := C.astikit_ftok(cpath, C.int(projectID), &errno)
	if key < 0 {
		return 0, fmt.Errorf("astikit: ftok failed: %s", syscall.Errno(errno))
	}
	return int(key), nil
}

const (
	IpcFlagCreat = int(C.IPC_CREAT)
	IpcFlagExcl  = int(C.IPC_EXCL)
)

type Semaphore struct {
	id  C.int
	key int
}

func newSemaphore(key, flags int) (*Semaphore, error) {
	// Get id
	var errno C.int
	id := C.astikit_sem_get(C.int(key), C.int(flags), &errno)
	if id < 0 {
		return nil, fmt.Errorf("astikit: sem_get failed: %s", syscall.Errno(errno))
	}
	return &Semaphore{
		id:  id,
		key: key,
	}, nil
}

func CreateSemaphore(key, flags int) (*Semaphore, error) {
	return newSemaphore(key, flags)
}

func OpenSemaphore(key int) (*Semaphore, error) {
	return newSemaphore(key, 0)
}

func (s *Semaphore) Close() error {
	// Already closed
	if s.id == -1 {
		return nil
	}

	// Close
	var errno C.int
	if ret := C.astikit_sem_close(s.id, &errno); ret < 0 {
		return fmt.Errorf("astikit: sem_close failed: %s", syscall.Errno(errno))
	}

	// Update
	s.id = -1
	s.key = -1
	return nil
}

func (s *Semaphore) Lock() error {
	// Closed
	if s.id == -1 {
		return errors.New("astikit: semaphore is closed")
	}

	// Lock
	var errno C.int
	ret := C.astikit_sem_lock(s.id, &errno)
	if ret < 0 {
		return fmt.Errorf("astikit: sem_lock failed: %s", syscall.Errno(errno))
	}
	return nil
}

func (s *Semaphore) Unlock() error {
	// Closed
	if s.id == -1 {
		return errors.New("astikit: semaphore is closed")
	}

	// Unlock
	var errno C.int
	ret := C.astikit_sem_unlock(s.id, &errno)
	if ret < 0 {
		return fmt.Errorf("astikit: sem_unlock failed: %s", syscall.Errno(errno))
	}
	return nil
}

func (s *Semaphore) Key() int {
	return s.key
}

type SharedMemory struct {
	addr unsafe.Pointer
	id   C.int
	key  int
}

func newSharedMemory(key, size, flags int) (w *SharedMemory, err error) {
	// Get id
	var errno C.int
	id := C.astikit_shm_get(C.int(key), C.int(size), C.int(flags), &errno)
	if id < 0 {
		err = fmt.Errorf("astikit: shm_get failed: %s", syscall.Errno(errno))
		return
	}

	// Create shared memory
	w = &SharedMemory{
		id:  id,
		key: key,
	}

	// Make sure to close shared memory in case of error
	defer func() {
		if err != nil {
			w.Close()
		}
	}()

	// Attach
	addr := C.astikit_shm_at(C.int(id), &errno)
	if addr == nil {
		err = fmt.Errorf("astikit: shm_at failed: %s", syscall.Errno(errno))
		return
	}

	// Update addr
	w.addr = addr
	return
}

func CreateSharedMemory(key, size, flags int) (*SharedMemory, error) {
	return newSharedMemory(key, size, flags)
}

func OpenSharedMemory(key int) (*SharedMemory, error) {
	return newSharedMemory(key, 0, 0)
}

func (w *SharedMemory) Close() error {
	// Already closed
	if w.id == -1 {
		return nil
	}

	// Close
	var errno C.int
	if ret := C.astikit_shm_close(w.id, w.addr, &errno); ret < 0 {
		return fmt.Errorf("astikit: shm_close failed: %s", syscall.Errno(errno))
	}

	// Update
	w.addr = nil
	w.id = -1
	w.key = -1
	return nil
}

func (w *SharedMemory) Write(src unsafe.Pointer, size int) error {
	// Closed
	if w.id == -1 {
		return errors.New("astikit: shared memory is closed")
	}

	// Copy
	C.memcpy(w.addr, src, C.ulong(size))
	return nil
}

func (w *SharedMemory) WriteBytes(b []byte) error {
	// Get c bytes
	cb := C.CBytes(b)
	defer C.free(cb)

	// Write
	return w.Write(cb, len(b))
}

func (w *SharedMemory) Pointer() unsafe.Pointer {
	return w.addr
}

func (w *SharedMemory) Key() int {
	return w.key
}

func (w *SharedMemory) ReadBytes(size int) ([]byte, error) {
	// Closed
	if w.id == -1 {
		return nil, errors.New("astikit: shared memory is closed")
	}

	// Get bytes
	return C.GoBytes(w.addr, C.int(size)), nil
}

type SemaphoredSharedMemoryWriter struct {
	id      string
	m       sync.Mutex // Locks write operations
	sem     *Semaphore
	semPath string
	shm     *SharedMemory
	shmPath string
	shmSize int
}

func NewSemaphoredSharedMemoryWriter(id string) *SemaphoredSharedMemoryWriter {
	return &SemaphoredSharedMemoryWriter{id: id}
}

func (w *SemaphoredSharedMemoryWriter) closeSemaphore() {
	if w.sem != nil {
		w.sem.Close()
	}
	if w.semPath != "" {
		os.RemoveAll(w.semPath)
	}
}

func (w *SemaphoredSharedMemoryWriter) closeSharedMemory() {
	if w.shm != nil {
		w.shm.Close()
	}
	if w.shmPath != "" {
		os.RemoveAll(w.shmPath)
	}
}

func (w *SemaphoredSharedMemoryWriter) Close() {
	w.closeSemaphore()
	w.closeSharedMemory()
}

func (w *SemaphoredSharedMemoryWriter) newSystemVIpcKey(id string) (key int, path string, err error) {
	// Make sure to clean in case of error
	var f *os.File
	defer func() {
		if err != nil {
			if f != nil {
				os.RemoveAll(f.Name())
			}
		}
	}()

	// Create temp file
	if f, err = os.CreateTemp("", w.id+id+"*"); err != nil {
		err = fmt.Errorf("astikit: creating temporary file failed: %w", err)
		return
	}
	f.Close() //nolint: errcheck

	// Update path
	path = f.Name()

	// Create key
	if key, err = NewSystemVIpcKey(1, path); err != nil {
		err = fmt.Errorf("astikit: creating system V ipc key failed: %w", err)
		return
	}
	return
}

func (w *SemaphoredSharedMemoryWriter) Write(src unsafe.Pointer, size int) (ro *SemaphoredSharedMemoryReadOptions, err error) {
	// Lock
	w.m.Lock()
	defer w.m.Unlock()

	// Shared memory has not been created or write size has changed, we need to allocate a new shared memory segment
	if w.shm == nil || size != w.shmSize {
		// Close previous shared memory
		w.closeSharedMemory()

		// Create shared memory key
		var shmKey int
		var shmPath string
		if shmKey, shmPath, err = w.newSystemVIpcKey("shm"); err != nil {
			err = fmt.Errorf("astikit: creating shared memory key failed: %w", err)
			return
		}

		// Create shared memory
		var shm *SharedMemory
		if shm, err = CreateSharedMemory(shmKey, size, IpcFlagCreat|IpcFlagExcl|0666); err != nil {
			err = fmt.Errorf("astikit: creating shared memory failed: %w", err)
			return
		}

		// Store attributes
		w.shm = shm
		w.shmPath = shmPath
		w.shmSize = size
	}

	// Semaphore has not been created
	if w.sem == nil {
		// Create semaphore key
		var semKey int
		var semPath string
		if semKey, semPath, err = w.newSystemVIpcKey("sem"); err != nil {
			err = fmt.Errorf("astikit: created semaphore key failed: %w", err)
			return
		}

		// Create semaphore
		var sem *Semaphore
		if sem, err = CreateSemaphore(semKey, IpcFlagCreat|IpcFlagExcl|0666); err != nil {
			err = fmt.Errorf("astikit: creating semaphore failed: %w", err)
			return
		}

		// Store attributes
		w.sem = sem
		w.semPath = semPath
	}

	// Lock
	if err = w.sem.Lock(); err != nil {
		err = fmt.Errorf("astikit: locking semaphore failed: %w", err)
		return
	}

	// Write
	if err = w.shm.Write(src, size); err != nil {
		err = fmt.Errorf("astikit: writing to shared memory failed: %w", err)
		return
	}

	// Unlock
	if err = w.sem.Unlock(); err != nil {
		err = fmt.Errorf("astikit: unlocking semaphore failed: %w", err)
		return
	}

	// Create read options
	ro = &SemaphoredSharedMemoryReadOptions{
		SemaphoreKey:    w.sem.Key(),
		SharedMemoryKey: w.shm.Key(),
		Size:            size,
	}
	return
}

func (w *SemaphoredSharedMemoryWriter) WriteBytes(b []byte) (*SemaphoredSharedMemoryReadOptions, error) {
	// Get c bytes
	cb := C.CBytes(b)
	defer C.free(cb)

	// Write
	return w.Write(cb, len(b))
}

type SemaphoredSharedMemoryReader struct {
	m   sync.Mutex // Locks read operations
	sem *Semaphore
	shm *SharedMemory
}

func NewSemaphoredSharedMemoryReader() *SemaphoredSharedMemoryReader {
	return &SemaphoredSharedMemoryReader{}
}

func (r *SemaphoredSharedMemoryReader) closeSemaphore() {
	if r.sem != nil {
		r.sem.Close()
	}
}

func (r *SemaphoredSharedMemoryReader) closeSharedMemory() {
	if r.shm != nil {
		r.shm.Close()
	}
}

func (r *SemaphoredSharedMemoryReader) Close() {
	r.closeSemaphore()
	r.closeSharedMemory()
}

type SemaphoredSharedMemoryReadOptions struct {
	SemaphoreKey    int
	SharedMemoryKey int
	Size            int
}

func (r *SemaphoredSharedMemoryReader) ReadBytes(o *SemaphoredSharedMemoryReadOptions) (b []byte, err error) {
	// Lock
	r.m.Lock()
	defer r.m.Unlock()

	// Shared memory is not opened or key has changed
	if r.shm == nil || r.shm.Key() != o.SharedMemoryKey {
		// Close previous shared memory
		r.closeSharedMemory()

		// Open shared memory
		var shm *SharedMemory
		if shm, err = OpenSharedMemory(o.SharedMemoryKey); err != nil {
			err = fmt.Errorf("astikit: opening shared memory failed: %w", err)
			return
		}

		// Store attributes
		r.shm = shm
	}

	// Semaphore is not opened or key has changed
	if r.sem == nil || r.sem.Key() != o.SemaphoreKey {
		// Close previous semaphore
		r.closeSemaphore()

		// Open semaphore
		var sem *Semaphore
		if sem, err = OpenSemaphore(o.SemaphoreKey); err != nil {
			err = fmt.Errorf("astikit: opening semaphore failed: %w", err)
			return
		}

		// Store attributes
		r.sem = sem
	}

	// Lock
	if err = r.sem.Lock(); err != nil {
		err = fmt.Errorf("astikit: locking semaphore failed: %w", err)
		return
	}

	// Copy
	b = make([]byte, o.Size)
	C.memcpy(unsafe.Pointer(&b[0]), r.shm.Pointer(), C.size_t(o.Size))

	// Unlock
	if err = r.sem.Unlock(); err != nil {
		err = fmt.Errorf("astikit: unlocking semaphore failed: %w", err)
		return
	}
	return
}
