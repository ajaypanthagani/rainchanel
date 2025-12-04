package dto

type Task struct {
	ID         uint   `json:"id"`
	WasmModule string `json:"wasm_module"`
	Func       string `json:"func"`
	Args       any    `json:"args"`
	CreatedBy  uint   `json:"created_by,omitempty"`
}
