//go:build !windows

package astikit

//#include <sys/shm.h>
//#include <sys/stat.h>
//#include <stdlib.h>
//#include <string.h>
//#include "ipc.h"
import "C"
import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

type PosixSharedMemory struct {
	addr   unsafe.Pointer
	cname  string
	fd     *C.int
	name   string
	size   int
	unlink bool
}

func newPosixSharedMemory(name string, flags, mode int, cb func(shm *PosixSharedMemory) error) (shm *PosixSharedMemory, err error) {
	// Create shared memory
	shm = &PosixSharedMemory{name: name}

	// To have a similar behavior with python, we need to handle the leading slash the same way:
	//   - make sure the "public" name has no leading "/"
	//   - make sure the "internal" name has a leading "/"
	shm.name = strings.TrimPrefix(shm.name, "/")
	shm.cname = "/" + shm.name

	// Get c name
	cname := C.CString(shm.cname)
	defer C.free(unsafe.Pointer(cname))

	// Get file descriptor
	var errno C.int
	fd := C.astikit_shm_open(cname, C.int(flags), C.mode_t(mode), &errno)
	if fd < 0 {
		err = fmt.Errorf("astikit: shm_open failed: %w", syscall.Errno(errno))
		return
	}
	shm.fd = &fd

	// Make sure to close shared memory in case of error
	defer func() {
		if err != nil {
			shm.Close()
		}
	}()

	// Callback
	if cb != nil {
		if err = cb(shm); err != nil {
			err = fmt.Errorf("astikit: callback failed: %w", err)
			return
		}
	}

	// Get size
	var stat C.struct_stat
	if ret := C.astikit_fstat(*shm.fd, &stat, &errno); ret < 0 {
		err = fmt.Errorf("astikit: fstat failed: %w", syscall.Errno(errno))
		return
	}
	shm.size = int(stat.st_size)

	// Map memory
	addr := C.astikit_mmap(C.size_t(shm.size), *shm.fd, &errno)
	if addr == nil {
		err = fmt.Errorf("astikit: mmap failed: %w", syscall.Errno(errno))
		return
	}

	// Update addr
	shm.addr = addr
	return
}

func CreatePosixSharedMemory(name string, size int) (*PosixSharedMemory, error) {
	return newPosixSharedMemory(name, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600, func(shm *PosixSharedMemory) (err error) {
		// Shared memory needs to be unlink on close
		shm.unlink = true

		// Truncate
		var errno C.int
		if ret := C.astikit_ftruncate(*shm.fd, C.off_t(size), &errno); ret < 0 {
			err = fmt.Errorf("astikit: ftruncate failed: %w", syscall.Errno(errno))
			return
		}
		return
	})
}

func OpenPosixSharedMemory(name string) (*PosixSharedMemory, error) {
	return newPosixSharedMemory(name, os.O_RDWR, 0600, nil)
}

func (shm *PosixSharedMemory) Close() error {
	// Unlink
	if shm.unlink {
		// Get c name
		cname := C.CString(shm.cname)
		defer C.free(unsafe.Pointer(cname))

		// Unlink
		var errno C.int
		if ret := C.astikit_shm_unlink(cname, &errno); ret < 0 {
			return fmt.Errorf("astikit: unlink failed: %w", syscall.Errno(errno))
		}
		shm.unlink = false
	}

	// Unmap memory
	if shm.addr != nil {
		var errno C.int
		if ret := C.astikit_munmap(shm.addr, C.size_t(shm.size), &errno); ret < 0 {
			return fmt.Errorf("astikit: munmap failed: %w", syscall.Errno(errno))
		}
		shm.addr = nil
	}

	// Close file descriptor
	if shm.fd != nil {
		var errno C.int
		if ret := C.astikit_close(*shm.fd, &errno); ret < 0 {
			return fmt.Errorf("astikit: close failed: %w", syscall.Errno(errno))
		}
		shm.fd = nil
	}
	return nil
}

func (shm *PosixSharedMemory) Write(src unsafe.Pointer, size int) error {
	// Unmapped
	if shm.addr == nil {
		return errors.New("astikit: shared memory is unmapped")
	}

	// Copy
	C.memcpy(shm.addr, src, C.size_t(size))
	return nil
}

func (shm *PosixSharedMemory) WriteBytes(b []byte) error {
	// Get c bytes
	cb := C.CBytes(b)
	defer C.free(cb)

	// Write
	return shm.Write(cb, len(b))
}

func (shm *PosixSharedMemory) ReadBytes(size int) ([]byte, error) {
	// Unmapped
	if shm.addr == nil {
		return nil, errors.New("astikit: shared memory is unmapped")
	}

	// Get bytes
	return C.GoBytes(shm.addr, C.int(size)), nil
}

func (shm *PosixSharedMemory) Name() string {
	return shm.name
}

func (shm *PosixSharedMemory) Size() int {
	return shm.size
}

func (shm *PosixSharedMemory) Addr() unsafe.Pointer {
	return shm.addr
}

type PosixVariableSizeSharedMemoryWriter struct {
	m      sync.Mutex // Locks write operations
	prefix string
	shm    *PosixSharedMemory
}

func NewPosixVariableSizeSharedMemoryWriter(prefix string) *PosixVariableSizeSharedMemoryWriter {
	return &PosixVariableSizeSharedMemoryWriter{prefix: prefix}
}

func (w *PosixVariableSizeSharedMemoryWriter) closeSharedMemory() {
	if w.shm != nil {
		w.shm.Close()
	}
}

func (w *PosixVariableSizeSharedMemoryWriter) Close() {
	w.closeSharedMemory()
}

func (w *PosixVariableSizeSharedMemoryWriter) Write(src unsafe.Pointer, size int) (ro PosixVariableSizeSharedMemoryReadOptions, err error) {
	// Lock
	w.m.Lock()
	defer w.m.Unlock()

	// Shared memory has not yet been created or previous shared memory segment is too small
	if w.shm == nil || size > w.shm.Size() {
		// Close previous shared memory
		w.closeSharedMemory()

		// Create shared memory
		var shm *PosixSharedMemory
		if shm, err = CreatePosixSharedMemory(w.prefix+"-"+strconv.Itoa(size), size); err != nil {
			err = fmt.Errorf("astikit: creating shared memory failed: %w", err)
			return
		}

		// Store shared memory
		w.shm = shm
	}

	// Write
	if err = w.shm.Write(src, size); err != nil {
		err = fmt.Errorf("astikit: writing to shared memory failed: %w", err)
		return
	}

	// Create read options
	ro = PosixVariableSizeSharedMemoryReadOptions{
		Name: w.shm.Name(),
		Size: size,
	}
	return
}

func (w *PosixVariableSizeSharedMemoryWriter) WriteBytes(b []byte) (PosixVariableSizeSharedMemoryReadOptions, error) {
	// Get c bytes
	cb := C.CBytes(b)
	defer C.free(cb)

	// Write
	return w.Write(cb, len(b))
}

type PosixVariableSizeSharedMemoryReader struct {
	m   sync.Mutex // Locks read operations
	shm *PosixSharedMemory
}

func NewPosixVariableSizeSharedMemoryReader() *PosixVariableSizeSharedMemoryReader {
	return &PosixVariableSizeSharedMemoryReader{}
}

func (r *PosixVariableSizeSharedMemoryReader) closeSharedMemory() {
	if r.shm != nil {
		r.shm.Close()
	}
}

func (r *PosixVariableSizeSharedMemoryReader) Close() {
	r.closeSharedMemory()
}

type PosixVariableSizeSharedMemoryReadOptions struct {
	Name string `json:"name"`
	Size int    `json:"size"`
}

func (r *PosixVariableSizeSharedMemoryReader) ReadBytes(o PosixVariableSizeSharedMemoryReadOptions) (b []byte, err error) {
	// Lock
	r.m.Lock()
	defer r.m.Unlock()

	// Shared memory has not yet been opened or shared memory's name has changed
	if r.shm == nil || r.shm.Name() != o.Name {
		// Close previous shared memory
		r.closeSharedMemory()

		// Open shared memory
		var shm *PosixSharedMemory
		if shm, err = OpenPosixSharedMemory(o.Name); err != nil {
			err = fmt.Errorf("astikit: opening shared memory failed: %w", err)
			return
		}

		// Store attributes
		r.shm = shm
	}

	// Copy
	b = make([]byte, o.Size)
	C.memcpy(unsafe.Pointer(&b[0]), r.shm.Addr(), C.size_t(o.Size))
	return
}

func NewSystemVKey(projectID int, path string) (int, error) {
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
	IpcCreate    = C.IPC_CREAT
	IpcExclusive = C.IPC_EXCL
)

type SystemVSemaphore struct {
	id  C.int
	key int
}

func newSystemVSemaphore(key int, flags int) (*SystemVSemaphore, error) {
	// Get id
	var errno C.int
	id := C.astikit_sem_get(C.int(key), C.int(flags), &errno)
	if id < 0 {
		return nil, fmt.Errorf("astikit: sem_get failed: %w", syscall.Errno(errno))
	}
	return &SystemVSemaphore{
		id:  id,
		key: key,
	}, nil
}

func CreateSystemVSemaphore(key, flags int) (*SystemVSemaphore, error) {
	return newSystemVSemaphore(key, flags)
}

func OpenSystemVSemaphore(key int) (*SystemVSemaphore, error) {
	return newSystemVSemaphore(key, 0)
}

func (s *SystemVSemaphore) Close() error {
	// Already closed
	if s.id == -1 {
		return nil
	}

	// Close
	var errno C.int
	if ret := C.astikit_sem_close(s.id, &errno); ret < 0 {
		return fmt.Errorf("astikit: sem_close failed: %w", syscall.Errno(errno))
	}

	// Update
	s.id = -1
	s.key = -1
	return nil
}

func (s *SystemVSemaphore) Lock() error {
	// Closed
	if s.id == -1 {
		return errors.New("astikit: semaphore is closed")
	}

	// Lock
	var errno C.int
	ret := C.astikit_sem_lock(s.id, &errno)
	if ret < 0 {
		return fmt.Errorf("astikit: sem_lock failed: %w", syscall.Errno(errno))
	}
	return nil
}

func (s *SystemVSemaphore) Unlock() error {
	// Closed
	if s.id == -1 {
		return errors.New("astikit: semaphore is closed")
	}

	// Unlock
	var errno C.int
	ret := C.astikit_sem_unlock(s.id, &errno)
	if ret < 0 {
		return fmt.Errorf("astikit: sem_unlock failed: %w", syscall.Errno(errno))
	}
	return nil
}

func (s *SystemVSemaphore) Key() int {
	return s.key
}

type SystemVSharedMemory struct {
	addr unsafe.Pointer
	id   C.int
	key  int
}

func newSystemVSharedMemory(key, size int, flags int) (shm *SystemVSharedMemory, err error) {
	// Get id
	var errno C.int
	id := C.astikit_shm_get(C.int(key), C.int(size), C.int(flags), &errno)
	if id < 0 {
		err = fmt.Errorf("astikit: shm_get failed: %w", syscall.Errno(errno))
		return
	}

	// Create shared memory
	shm = &SystemVSharedMemory{
		id:  id,
		key: key,
	}

	// Make sure to close shared memory in case of error
	defer func() {
		if err != nil {
			shm.Close()
		}
	}()

	// Attach
	addr := C.astikit_shm_at(C.int(id), &errno)
	if addr == nil {
		err = fmt.Errorf("astikit: shm_at failed: %w", syscall.Errno(errno))
		return
	}

	// Update addr
	shm.addr = addr
	return
}

func CreateSystemVSharedMemory(key, size, flags int) (*SystemVSharedMemory, error) {
	return newSystemVSharedMemory(key, size, flags)
}

func OpenSystemVSharedMemory(key int) (*SystemVSharedMemory, error) {
	return newSystemVSharedMemory(key, 0, 0)
}

func (shm *SystemVSharedMemory) Close() error {
	// Already closed
	if shm.id == -1 {
		return nil
	}

	// Close
	var errno C.int
	if ret := C.astikit_shm_close(shm.id, shm.addr, &errno); ret < 0 {
		return fmt.Errorf("astikit: shm_close failed: %w", syscall.Errno(errno))
	}

	// Update
	shm.addr = nil
	shm.id = -1
	shm.key = -1
	return nil
}

func (shm *SystemVSharedMemory) Write(src unsafe.Pointer, size int) error {
	// Closed
	if shm.id == -1 {
		return errors.New("astikit: shared memory is closed")
	}

	// Copy
	C.memcpy(shm.addr, src, C.size_t(size))
	return nil
}

func (shm *SystemVSharedMemory) WriteBytes(b []byte) error {
	// Get c bytes
	cb := C.CBytes(b)
	defer C.free(cb)

	// Write
	return shm.Write(cb, len(b))
}

func (shm *SystemVSharedMemory) Addr() unsafe.Pointer {
	return shm.addr
}

func (shm *SystemVSharedMemory) Key() int {
	return shm.key
}

func (shm *SystemVSharedMemory) ReadBytes(size int) ([]byte, error) {
	// Closed
	if shm.id == -1 {
		return nil, errors.New("astikit: shared memory is closed")
	}

	// Get bytes
	return C.GoBytes(shm.addr, C.int(size)), nil
}

type SystemVSemaphoredSharedMemoryWriter struct {
	m       sync.Mutex // Locks write operations
	sem     *SystemVSemaphore
	shm     *SystemVSharedMemory
	shmAt   int64
	shmSize int
}

func NewSystemVSemaphoredSharedMemoryWriter() *SystemVSemaphoredSharedMemoryWriter {
	return &SystemVSemaphoredSharedMemoryWriter{}
}

func (w *SystemVSemaphoredSharedMemoryWriter) closeSemaphore() {
	if w.sem != nil {
		w.sem.Close()
	}
}

func (w *SystemVSemaphoredSharedMemoryWriter) closeSharedMemory() {
	if w.shm != nil {
		w.shm.Close()
	}
}

func (w *SystemVSemaphoredSharedMemoryWriter) Close() {
	w.closeSemaphore()
	w.closeSharedMemory()
}

func (w *SystemVSemaphoredSharedMemoryWriter) generateRandomKey(f func(key int) error) error {
	try := 0
	for {
		key := int(int32(randSrc.Int63()))
		if key == int(C.IPC_PRIVATE) {
			continue
		}
		err := f(key)
		if errors.Is(err, syscall.EEXIST) {
			if try++; try < 10000 {
				continue
			}
			return errors.New("astikit: max tries reached")
		}
		return err
	}
}

func (w *SystemVSemaphoredSharedMemoryWriter) Write(src unsafe.Pointer, size int) (ro *SystemVSemaphoredSharedMemoryReadOptions, err error) {
	// Lock
	w.m.Lock()
	defer w.m.Unlock()

	// Shared memory has not been created or previous shared memory segment is too small,
	// we need to allocate a new shared memory segment
	if w.shm == nil || size > w.shmSize {
		// Close previous shared memory
		w.closeSharedMemory()

		// Generate random key
		if err = w.generateRandomKey(func(key int) (err error) {
			// Create shared memory
			var shm *SystemVSharedMemory
			if shm, err = CreateSystemVSharedMemory(key, size, IpcCreate|IpcExclusive|0666); err != nil {
				err = fmt.Errorf("astikit: creating shared memory failed: %w", err)
				return
			}

			// Store attributes
			w.shm = shm
			w.shmAt = time.Now().UnixNano()
			w.shmSize = size
			return
		}); err != nil {
			err = fmt.Errorf("astikit: generating random key failed: %w", err)
			return
		}
	}

	// Semaphore has not been created
	if w.sem == nil {
		// Generate random key
		if err = w.generateRandomKey(func(key int) (err error) {
			// Create semaphore
			var sem *SystemVSemaphore
			if sem, err = CreateSystemVSemaphore(key, IpcCreate|IpcExclusive|0666); err != nil {
				err = fmt.Errorf("astikit: creating semaphore failed: %w", err)
				return
			}

			// Store attributes
			w.sem = sem
			return
		}); err != nil {
			err = fmt.Errorf("astikit: generating random key failed: %w", err)
			return
		}
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
	ro = &SystemVSemaphoredSharedMemoryReadOptions{
		SemaphoreKey:    w.sem.Key(),
		SharedMemoryAt:  w.shmAt,
		SharedMemoryKey: w.shm.Key(),
		Size:            size,
	}
	return
}

func (w *SystemVSemaphoredSharedMemoryWriter) WriteBytes(b []byte) (*SystemVSemaphoredSharedMemoryReadOptions, error) {
	// Get c bytes
	cb := C.CBytes(b)
	defer C.free(cb)

	// Write
	return w.Write(cb, len(b))
}

type SystemVSemaphoredSharedMemoryReader struct {
	m     sync.Mutex // Locks read operations
	sem   *SystemVSemaphore
	shm   *SystemVSharedMemory
	shmAt int64
}

func NewSystemVSemaphoredSharedMemoryReader() *SystemVSemaphoredSharedMemoryReader {
	return &SystemVSemaphoredSharedMemoryReader{}
}

func (r *SystemVSemaphoredSharedMemoryReader) closeSemaphore() {
	if r.sem != nil {
		r.sem.Close()
	}
}

func (r *SystemVSemaphoredSharedMemoryReader) closeSharedMemory() {
	if r.shm != nil {
		r.shm.Close()
	}
}

func (r *SystemVSemaphoredSharedMemoryReader) Close() {
	r.closeSemaphore()
	r.closeSharedMemory()
}

type SystemVSemaphoredSharedMemoryReadOptions struct {
	SemaphoreKey    int
	SharedMemoryAt  int64
	SharedMemoryKey int
	Size            int
}

func (r *SystemVSemaphoredSharedMemoryReader) ReadBytes(o *SystemVSemaphoredSharedMemoryReadOptions) (b []byte, err error) {
	// Lock
	r.m.Lock()
	defer r.m.Unlock()

	// Shared memory is not opened or shared memory has changed
	if r.shm == nil || r.shm.Key() != o.SharedMemoryKey || r.shmAt != o.SharedMemoryAt {
		// Close previous shared memory
		r.closeSharedMemory()

		// Open shared memory
		var shm *SystemVSharedMemory
		if shm, err = OpenSystemVSharedMemory(o.SharedMemoryKey); err != nil {
			err = fmt.Errorf("astikit: opening shared memory failed: %w", err)
			return
		}

		// Store attributes
		r.shm = shm
		r.shmAt = o.SharedMemoryAt
	}

	// Semaphore is not opened
	if r.sem == nil {
		// Close previous semaphore
		r.closeSemaphore()

		// Open semaphore
		var sem *SystemVSemaphore
		if sem, err = OpenSystemVSemaphore(o.SemaphoreKey); err != nil {
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
	C.memcpy(unsafe.Pointer(&b[0]), r.shm.Addr(), C.size_t(o.Size))

	// Unlock
	if err = r.sem.Unlock(); err != nil {
		err = fmt.Errorf("astikit: unlocking semaphore failed: %w", err)
		return
	}
	return
}
