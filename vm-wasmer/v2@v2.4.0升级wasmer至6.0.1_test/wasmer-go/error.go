package wasmer

// #include <wasmer.h>
import "C"

import (
	"runtime"
	"unsafe"
)

// Error represents a Wasmer runtime error.
type Error struct {
	message string
}

func newErrorWith(message string) *Error {
	return &Error{
		message: message,
	}
}

func _newErrorFromWasmer() *Error {
	errorLength := C.wasmer_last_error_length()

	if errorLength == 0 {
		return newErrorWith("(no error from Wasmer)")
	}

	errorMessage := make([]C.char, errorLength)
	errorMessagePointer := (*C.char)(unsafe.Pointer(&errorMessage[0]))

	errorResult := C.wasmer_last_error_message(errorMessagePointer, errorLength)

	if errorResult == -1 {
		return newErrorWith("(failed to read last error from Wasmer)")
	}

	return newErrorWith(C.GoStringN(errorMessagePointer, errorLength-1))
}

func maybeNewErrorFromWasmer(block func() bool) *Error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if block() /* has failed */ {
		return _newErrorFromWasmer()
	}

	return nil
}

// Error returns the Error's message.
func (error *Error) Error() string {
	return error.message
}

// TrapError represents a trap produced during Wasm execution.
//
// # See also
//
// Specification: https://webassembly.github.io/spec/core/intro/overview.html#trap
type TrapError struct {
	message string
	origin  *Frame
	trace   []*Frame
}

func newErrorFromTrap(pointer *C.wasm_trap_t) *TrapError {
	trap := newTrapFromPointer(pointer, nil)
	//fmt.Println("hihihihi newErrorFromTrap")
	//fmt.Println(trap.Message())
	//fmt.Println("end newErrorFromTrap")
	//fmt.Printf("trap.Trace().frames %+v\n", trap.Trace().frames)
	//fmt.Printf("trap.Origin() %+v\n", trap.Origin())
	return &TrapError{
		message: trap.Message(),
		origin:  trap.Origin(),
		trace:   trap.Trace().frames,
	}
}

// Error returns the TrapError's message.
func (self *TrapError) Error() string {
	return self.message
}

// Origin returns the TrapError's origin as a Frame.
func (self *TrapError) Origin() *Frame {
	return self.origin
}

// Trace returns the TrapError's trace as a Frame array.
func (self *TrapError) Trace() []*Frame {
	return self.trace
}
