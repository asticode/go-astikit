//go:build !windows

package astiposix

//#include <stdlib.h>
//#include <string.h>
//#include "posix.h"
import "C"
import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"unsafe"
)

type SharedMemory struct {
	addr   unsafe.Pointer
	cname  string
	fd     *C.int
	name   string
	size   int
	unlink bool
}

func newSharedMemory(name string, flags, mode int, cb func(shm *SharedMemory) error) (shm *SharedMemory, err error) {
	// Create shared memory
	shm = &SharedMemory{name: name}

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

func CreateSharedMemory(name string, size int) (*SharedMemory, error) {
	return newSharedMemory(name, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600, func(shm *SharedMemory) (err error) {
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

func OpenSharedMemory(name string) (*SharedMemory, error) {
	return newSharedMemory(name, os.O_RDWR, 0600, nil)
}

func (shm *SharedMemory) Close() error {
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

func (shm *SharedMemory) Write(src unsafe.Pointer, size int) error {
	// Unmapped
	if shm.addr == nil {
		return errors.New("astikit: shared memory is unmapped")
	}

	// Copy
	C.memcpy(shm.addr, src, C.size_t(size))
	return nil
}

func (shm *SharedMemory) WriteBytes(b []byte) error {
	// Get c bytes
	cb := C.CBytes(b)
	defer C.free(cb)

	// Write
	return shm.Write(cb, len(b))
}

func (shm *SharedMemory) ReadBytes(size int) ([]byte, error) {
	// Unmapped
	if shm.addr == nil {
		return nil, errors.New("astikit: shared memory is unmapped")
	}

	// Get bytes
	return C.GoBytes(shm.addr, C.int(size)), nil
}

func (shm *SharedMemory) Name() string {
	return shm.name
}

func (shm *SharedMemory) Size() int {
	return shm.size
}

func (shm *SharedMemory) Addr() unsafe.Pointer {
	return shm.addr
}

type VariableSizeSharedMemoryWriter struct {
	m      sync.Mutex // Locks write operations
	prefix string
	shm    *SharedMemory
}

func NewVariableSizeSharedMemoryWriter(prefix string) *VariableSizeSharedMemoryWriter {
	return &VariableSizeSharedMemoryWriter{prefix: prefix}
}

func (w *VariableSizeSharedMemoryWriter) closeSharedMemory() {
	if w.shm != nil {
		w.shm.Close()
	}
}

func (w *VariableSizeSharedMemoryWriter) Close() {
	w.closeSharedMemory()
}

func (w *VariableSizeSharedMemoryWriter) Write(src unsafe.Pointer, size int) (ro VariableSizeSharedMemoryReadOptions, err error) {
	// Lock
	w.m.Lock()
	defer w.m.Unlock()

	// Shared memory has not yet been created or previous shared memory segment is too small
	if w.shm == nil || size > w.shm.Size() {
		// Close previous shared memory
		w.closeSharedMemory()

		// Create shared memory
		var shm *SharedMemory
		if shm, err = CreateSharedMemory(w.prefix+"-"+strconv.Itoa(size), size); err != nil {
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
	ro = VariableSizeSharedMemoryReadOptions{
		Name: w.shm.Name(),
		Size: size,
	}
	return
}

func (w *VariableSizeSharedMemoryWriter) WriteBytes(b []byte) (VariableSizeSharedMemoryReadOptions, error) {
	// Get c bytes
	cb := C.CBytes(b)
	defer C.free(cb)

	// Write
	return w.Write(cb, len(b))
}

type VariableSizeSharedMemoryReader struct {
	m   sync.Mutex // Locks read operations
	shm *SharedMemory
}

func NewVariableSizeSharedMemoryReader() *VariableSizeSharedMemoryReader {
	return &VariableSizeSharedMemoryReader{}
}

func (r *VariableSizeSharedMemoryReader) closeSharedMemory() {
	if r.shm != nil {
		r.shm.Close()
	}
}

func (r *VariableSizeSharedMemoryReader) Close() {
	r.closeSharedMemory()
}

type VariableSizeSharedMemoryReadOptions struct {
	Name string `json:"name"`
	Size int    `json:"size"`
}

func (r *VariableSizeSharedMemoryReader) ReadBytes(o VariableSizeSharedMemoryReadOptions) (b []byte, err error) {
	// Lock
	r.m.Lock()
	defer r.m.Unlock()

	// Shared memory has not yet been opened or shared memory's name has changed
	if r.shm == nil || r.shm.Name() != o.Name {
		// Close previous shared memory
		r.closeSharedMemory()

		// Open shared memory
		var shm *SharedMemory
		if shm, err = OpenSharedMemory(o.Name); err != nil {
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
