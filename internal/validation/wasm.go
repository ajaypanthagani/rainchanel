package validation

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

var (
	ErrInvalidWASMModule     = errors.New("invalid WASM module")
	ErrFunctionNotExported   = errors.New("function is not exported")
	ErrInvalidFunctionArgs   = errors.New("function arguments do not match signature")
	ErrInvalidBase64Encoding = errors.New("invalid base64 encoding for WASM module")
)

func ValidateTask(wasmModuleBase64, functionName string, args interface{}) error {
	wasmBytes, err := base64.StdEncoding.DecodeString(wasmModuleBase64)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidBase64Encoding, err)
	}

	exportedFunc, exportedNames, err := validateWASMModuleAndGetFunction(wasmBytes, functionName)
	if err != nil {
		return err
	}

	if err := validateFunctionSignature(exportedFunc, args); err != nil {
		return err
	}

	_ = exportedNames
	return nil
}

func validateWASMModuleAndGetFunction(wasmBytes []byte, functionName string) (api.FunctionDefinition, []string, error) {
	ctx := context.Background()
	runtime := wazero.NewRuntime(ctx)
	defer runtime.Close(ctx)

	exportedFunctionNames, err := parseExportedFunctions(wasmBytes)
	if err != nil {
		_, compileErr := runtime.CompileModule(ctx, wasmBytes)
		if compileErr != nil {
			return nil, nil, fmt.Errorf("%w: %v", ErrInvalidWASMModule, compileErr)
		}

		exportedFunctionNames, err = getExportedFunctionNamesFromModule(ctx, runtime, wasmBytes)
		if err != nil {
			return nil, nil, fmt.Errorf("%w: %v", ErrInvalidWASMModule, err)
		}
	}

	userExportedFunctions := filterUserExportedFunctions(exportedFunctionNames)

	found := false
	for _, name := range userExportedFunctions {
		if name == functionName {
			found = true
			break
		}
	}

	if !found {
		if len(userExportedFunctions) > 0 {
			return nil, userExportedFunctions, fmt.Errorf("%w: function '%s' not found. Available exported functions: %v",
				ErrFunctionNotExported, functionName, userExportedFunctions)
		}
		return nil, userExportedFunctions, fmt.Errorf("%w: function '%s' not found in WASM module exports",
			ErrFunctionNotExported, functionName)
	}

	compiled, err := runtime.CompileModule(ctx, wasmBytes)
	if err != nil {
		return nil, userExportedFunctions, fmt.Errorf("%w: %v", ErrInvalidWASMModule, err)
	}

	module, err := runtime.InstantiateModule(ctx, compiled, wazero.NewModuleConfig())
	if err != nil {
		return nil, userExportedFunctions, fmt.Errorf("%w: failed to instantiate module: %v", ErrInvalidWASMModule, err)
	}
	defer module.Close(ctx)

	exportedFunc := module.ExportedFunction(functionName)
	if exportedFunc == nil {
		return nil, userExportedFunctions, fmt.Errorf("%w: function '%s' not accessible", ErrFunctionNotExported, functionName)
	}

	funcDef := exportedFunc.Definition()
	return funcDef, userExportedFunctions, nil
}

func parseExportedFunctions(wasmBytes []byte) ([]string, error) {
	if len(wasmBytes) < 8 {
		return nil, fmt.Errorf("invalid WASM binary: too short")
	}

	if string(wasmBytes[0:4]) != "\x00asm" {
		return nil, fmt.Errorf("invalid WASM magic number")
	}

	version := binary.LittleEndian.Uint32(wasmBytes[4:8])
	if version != 1 {
		return nil, fmt.Errorf("unsupported WASM version: %d", version)
	}

	var exportedFunctions []string
	pos := 8

	for pos < len(wasmBytes) {
		if pos >= len(wasmBytes) {
			break
		}

		sectionID := wasmBytes[pos]
		pos++

		if pos >= len(wasmBytes) {
			break
		}

		size, bytesRead := readULEB128(wasmBytes[pos:])
		if bytesRead == 0 {
			break
		}
		pos += bytesRead

		if sectionID == 7 {
			if pos >= len(wasmBytes) {
				break
			}
			exportCount, bytesRead := readULEB128(wasmBytes[pos:])
			if bytesRead == 0 {
				break
			}
			pos += bytesRead

			for i := uint64(0); i < exportCount && pos < len(wasmBytes); i++ {
				if pos >= len(wasmBytes) {
					break
				}
				nameLen, bytesRead := readULEB128(wasmBytes[pos:])
				if bytesRead == 0 || pos+bytesRead > len(wasmBytes) {
					break
				}
				pos += bytesRead

				if pos+int(nameLen) > len(wasmBytes) {
					break
				}

				exportName := string(wasmBytes[pos : pos+int(nameLen)])
				pos += int(nameLen)

				if pos >= len(wasmBytes) {
					break
				}

				exportKind := wasmBytes[pos]
				pos++

				if pos >= len(wasmBytes) {
					break
				}

				_, bytesRead = readULEB128(wasmBytes[pos:])
				if bytesRead == 0 || pos+bytesRead > len(wasmBytes) {
					break
				}
				pos += bytesRead

				if exportKind == 0 {
					exportedFunctions = append(exportedFunctions, exportName)
				}
			}
			break
		} else {
			if pos+int(size) > len(wasmBytes) {
				break
			}
			pos += int(size)
		}
	}

	return exportedFunctions, nil
}

func readULEB128(data []byte) (uint64, int) {
	var result uint64
	var shift uint
	var bytesRead int

	for bytesRead < len(data) {
		b := data[bytesRead]
		bytesRead++

		result |= uint64(b&0x7F) << shift
		if b&0x80 == 0 {
			break
		}
		shift += 7
		if shift >= 64 {
			return 0, 0
		}
	}

	return result, bytesRead
}

func filterUserExportedFunctions(allExports []string) []string {
	userExports := []string{}
	internalPrefixes := []string{"runtime.", "__"}
	internalNames := map[string]bool{
		"_start":                   true,
		"__wasm_call_ctors":        true,
		"__wasm_apply_data_relocs": true,
		"__wasm_init_memory":       true,
	}

	for _, name := range allExports {
		if internalNames[name] {
			continue
		}

		skip := false
		for _, prefix := range internalPrefixes {
			if strings.HasPrefix(name, prefix) {
				skip = true
				break
			}
		}

		if !skip {
			userExports = append(userExports, name)
		}
	}

	return userExports
}

func getExportedFunctionNamesFromModule(ctx context.Context, runtime wazero.Runtime, wasmBytes []byte) ([]string, error) {
	compiled, err := runtime.CompileModule(ctx, wasmBytes)
	if err != nil {
		return nil, err
	}

	module, err := runtime.InstantiateModule(ctx, compiled, wazero.NewModuleConfig())
	if err != nil {
		return nil, err
	}
	defer module.Close(ctx)

	var names []string
	return names, nil
}

func validateFunctionSignature(function api.FunctionDefinition, args interface{}) error {
	paramTypes := function.ParamTypes()
	paramCount := len(paramTypes)

	argsSlice, err := convertArgsToSlice(args)
	if err != nil {
		return fmt.Errorf("%w: failed to parse arguments: %v", ErrInvalidFunctionArgs, err)
	}

	argsCount := len(argsSlice)

	if paramCount != argsCount {
		return fmt.Errorf("%w: expected %d parameters, got %d",
			ErrInvalidFunctionArgs, paramCount, argsCount)
	}

	for i, paramType := range paramTypes {
		if i >= len(argsSlice) {
			break
		}

		arg := argsSlice[i]
		if err := validateArgType(arg, paramType); err != nil {
			return fmt.Errorf("%w: parameter %d: %v", ErrInvalidFunctionArgs, i, err)
		}
	}

	return nil
}

func convertArgsToSlice(args interface{}) ([]interface{}, error) {
	if args == nil {
		return []interface{}{}, nil
	}

	switch v := args.(type) {
	case []interface{}:
		return v, nil
	case []json.Number:
		result := make([]interface{}, len(v))
		for i, n := range v {
			result[i] = n
		}
		return result, nil
	case []float64:
		result := make([]interface{}, len(v))
		for i, n := range v {
			result[i] = n
		}
		return result, nil
	case []int:
		result := make([]interface{}, len(v))
		for i, n := range v {
			result[i] = n
		}
		return result, nil
	case []int32:
		result := make([]interface{}, len(v))
		for i, n := range v {
			result[i] = n
		}
		return result, nil
	case []int64:
		result := make([]interface{}, len(v))
		for i, n := range v {
			result[i] = n
		}
		return result, nil
	default:
		jsonBytes, err := json.Marshal(args)
		if err != nil {
			return nil, fmt.Errorf("cannot convert args to slice: %v", err)
		}

		var jsonArray []interface{}
		if err := json.Unmarshal(jsonBytes, &jsonArray); err == nil {
			return jsonArray, nil
		}

		return []interface{}{args}, nil
	}
}

func validateArgType(arg interface{}, expectedType api.ValueType) error {
	switch expectedType {
	case api.ValueTypeI32:
		switch v := arg.(type) {
		case int, int32, int64, float64:
			return nil
		case json.Number:
			_, err := v.Int64()
			return err
		case string:

			var num json.Number
			if err := json.Unmarshal([]byte(`"`+v+`"`), &num); err == nil {
				_, err := num.Int64()
				return err
			}
			return fmt.Errorf("cannot convert string to i32")
		default:
			return fmt.Errorf("cannot convert %T to i32", arg)
		}
	case api.ValueTypeI64:
		switch v := arg.(type) {
		case int, int32, int64, float64:
			return nil
		case json.Number:
			_, err := v.Int64()
			return err
		case string:
			var num json.Number
			if err := json.Unmarshal([]byte(`"`+v+`"`), &num); err == nil {
				_, err := num.Int64()
				return err
			}
			return fmt.Errorf("cannot convert string to i64")
		default:
			return fmt.Errorf("cannot convert %T to i64", arg)
		}
	case api.ValueTypeF32:
		switch v := arg.(type) {
		case float32, float64, int, int32, int64:
			return nil
		case json.Number:
			_, err := v.Float64()
			return err
		case string:
			var num json.Number
			if err := json.Unmarshal([]byte(`"`+v+`"`), &num); err == nil {
				_, err := num.Float64()
				return err
			}
			return fmt.Errorf("cannot convert string to f32")
		default:
			return fmt.Errorf("cannot convert %T to f32", arg)
		}
	case api.ValueTypeF64:
		switch v := arg.(type) {
		case float32, float64, int, int32, int64:
			return nil
		case json.Number:
			_, err := v.Float64()
			return err
		case string:
			var num json.Number
			if err := json.Unmarshal([]byte(`"`+v+`"`), &num); err == nil {
				_, err := num.Float64()
				return err
			}
			return fmt.Errorf("cannot convert string to f64")
		default:
			return fmt.Errorf("cannot convert %T to f64", arg)
		}
	default:
		return fmt.Errorf("unsupported WASM value type: %v", expectedType)
	}
}
