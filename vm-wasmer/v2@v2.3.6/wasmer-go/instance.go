package wasmer

// #include <wasmer.h>
import "C"
import (
	"fmt"
	"runtime"
)

type Instance struct {
	_inner  *C.wasm_instance_t
	Exports *Exports

	// without this, imported functions may be freed before execution of an exported function is complete.
	imports *ImportObject
}

// NewInstance instantiates a new Instance.
//
// It takes two arguments, the Module and an ImportObject.
//
// Note:️ Instantiating a module may return TrapError if the module's
// start function traps.
//
//   wasmBytes := []byte(`...`)
//   engine := wasmer.NewEngine()
//   store := wasmer.NewStore(engine)
//   module, err := wasmer.NewModule(store, wasmBytes)
//   importObject := wasmer.NewImportObject()
//   instance, err := wasmer.NewInstance(module, importObject)
//
func NewInstance(module *Module, imports *ImportObject) (*Instance, error) {
	var traps *C.wasm_trap_t
	externs, err := imports.intoInner(module)
	if err != nil {
		return nil, err
	}

	var instance *C.wasm_instance_t

	err2 := maybeNewErrorFromWasmer(func() bool {
		instance = C.wasm_instance_new(
			module.store.inner(),
			module.inner(),
			externs,
			&traps,
		)

		return traps == nil && instance == nil
	})

	if err2 != nil {
		return nil, err2
	}

	if traps != nil {
		return nil, newErrorFromTrap(traps)
	}

	self := &Instance{
		_inner:  instance,
		Exports: newExports(instance, module),
		imports: imports,
	}

	runtime.SetFinalizer(self, func(self *Instance) {
		self.Close()
	})

	return self, nil
}

func (self *Instance) inner() *C.wasm_instance_t {
	return self._inner
}

// Force to close the Instance.
//
// A runtime finalizer is registered on the Instance, but it is
// possible to force the destruction of the Instance by calling Close
// manually.
func (self *Instance) Close() {
	runtime.SetFinalizer(self, nil)
	if self.Exports != nil {
		self.Exports.Close()
	}
	if self.imports != nil {
		self.imports.Close()
	}
	C.wasm_instance_delete(self.inner())
	self.Exports.Close()
}

func (self *Instance) GetGasRemaining() uint64 {
	gasRemainingGlobal, err := self.Exports.GetGlobal("wasmer_metering_remaining_points")
	if err != nil {
		panic(fmt.Sprintf("can't find 'wasmer_metering_remaining_points' global variable: %v", err))
	}
	//fmt.Sprintf("%v", gasRemainingGlobal)

	gasRemaining, err := gasRemainingGlobal.Get()
	if err != nil {
		panic("can't get data from 'wasmer_metering_remaining_points' global variable.")
	}

	return uint64(gasRemaining.(int64))
}

func (self *Instance) SetGasLimit(gas uint64) {
	gasRemainingGlobal, err := self.Exports.GetGlobal("wasmer_metering_remaining_points")
	if err != nil {
		panic(fmt.Sprintf("can't find 'wasmer_metering_remaining_points' global variable: %v", err))
	}

	if gasRemainingGlobal.Set(int64(gas), I64) != nil {
		panic("can't get data from 'wasmer_metering_remaining_points' global variable.")
	}
	//fmt.Sprintf("%v", gasRemainingGlobal)
}
