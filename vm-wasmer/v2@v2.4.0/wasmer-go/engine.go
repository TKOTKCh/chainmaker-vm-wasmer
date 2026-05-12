package wasmer

// #include <wasmer.h>
import "C"
import "runtime"

// Engine is used by the Store to drive the compilation and the
// execution of a WebAssembly module.
type Engine struct {
	_inner *C.wasm_engine_t
}

func newEngine(engine *C.wasm_engine_t) *Engine {
	self := &Engine{
		_inner: engine,
	}

	runtime.SetFinalizer(self, func(self *Engine) {
		C.wasm_engine_delete(self.inner())
	})

	return self
}

// NewEngine instantiates and returns a new Engine with the default configuration.
//
//	engine := NewEngine()
func NewEngine() *Engine {
	return newEngine(C.wasm_engine_new())
}

// NewEngineWithConfig instantiates and returns a new Engine with the given configuration.
//
//	config := NewConfig()
//	engine := NewEngineWithConfig(config)
func NewEngineWithConfig(config *Config) *Engine {
	return newEngine(C.wasm_engine_new_with_config(config.inner()))
}

func (self *Engine) inner() *C.wasm_engine_t {
	return self._inner
}

// Close the engine forcingly
func (self *Engine) Close() {
	runtime.SetFinalizer(self, nil)
	C.wasm_engine_delete(self.inner())
}
