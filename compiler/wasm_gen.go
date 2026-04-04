package compiler

import (
	"fmt"
	"io"
	"strings"
)

// WasmGenerator generates JavaScript/WASM bindings for Ella services
type WasmGenerator struct {
	program         *Program
	packageName     string
	enums           map[string]*DeclEnum
	models          map[string]*DeclModel
	allowExtensions bool
}

// NewWasmGenerator creates a new WASM code generator
func NewWasmGenerator(program *Program, packageName string, allowExtensions bool) *WasmGenerator {
	g := &WasmGenerator{
		program:         program,
		packageName:     packageName,
		enums:           make(map[string]*DeclEnum),
		models:          make(map[string]*DeclModel),
		allowExtensions: allowExtensions,
	}

	// Pre-process to collect enums and models for type resolution
	for _, node := range program.Nodes {
		switch n := node.(type) {
		case *DeclEnum:
			g.enums[n.Name.Name] = n
		case *DeclModel:
			g.models[n.Name.Name] = n
		}
	}

	return g
}

// GenerateToWriter writes the WASM bindings to the writer
func (g *WasmGenerator) GenerateToWriter(w io.Writer) error {
	var sb strings.Builder

	// Write build tag
	sb.WriteString("//go:build js && wasm\n\n")

	// Write package
	sb.WriteString("package ")
	sb.WriteString(g.packageName)
	sb.WriteString("\n\n")

	// Collect services
	var services []*DeclService
	for _, node := range g.program.Nodes {
		if svc, ok := node.(*DeclService); ok {
			services = append(services, svc)
		}
	}

	if len(services) == 0 {
		// No services to generate
		sb.WriteString("// No services found\n")
		_, err := w.Write([]byte(sb.String()))
		return err
	}

	// Write imports
	sb.WriteString("import (\n")
	sb.WriteString("\t\"context\"\n")
	sb.WriteString("\t\"encoding/json\"\n")
	sb.WriteString("\t\"fmt\"\n")
	sb.WriteString("\t\"sync\"\n")
	sb.WriteString("\t\"syscall/js\"\n")
	sb.WriteString("\t\"time\"\n\n")
	sb.WriteString("\t\"ella.to/jsonrpc\"\n")
	sb.WriteString(")\n\n")

	// Write constants
	sb.WriteString("const (\n")
	sb.WriteString("\tellaObjName      = \"ella\"\n")
	sb.WriteString("\tellaEventReady   = \"ellaReady\"\n")
	sb.WriteString(")\n\n")

	// Write helper functions
	sb.WriteString(wasmHelperFunctions)

	// Write caller creation functions
	g.generateCallerFunctions(&sb)

	// Write service registration for each service
	for _, svc := range services {
		g.generateServiceWasmBindings(&sb, svc)
	}

	// Write main registration function
	g.generateMainRegistration(&sb, services)

	_, err := w.Write([]byte(sb.String()))
	return err
}

// Generate produces WASM binding source code
func (g *WasmGenerator) Generate() (string, error) {
	var sb strings.Builder

	if err := g.GenerateToWriter(&sb); err != nil {
		return "", err
	}

	return sb.String(), nil
}

func (g *WasmGenerator) generateCallerFunctions(sb *strings.Builder) {
	sb.WriteString(`
// createHttpCaller creates and stores an HTTP caller, returning a handler ID.
func createHttpCaller(host string, withTrace bool, headers map[string]string) int {
	client := jsonrpc.NewHTTPClient(host, jsonrpc.WithTrace(withTrace))
	for k, v := range headers {
		client.WithHeader(k, v)
	}

	handlerID := getNextHandlerID()

	handlersMutex.Lock()
	handlersMap[handlerID] = client
	handlersMutex.Unlock()

	return handlerID
}

// jsCreateHttpCaller is the JavaScript-callable function to create an HTTP caller.
// Call as: ella.createHttpCaller("https://api.example.com", true, {"Authorization": "Bearer token"})
func jsCreateHttpCaller(_ js.Value, args []js.Value) any {
	host, ok := jsGetStringArg(args, 0)
	if !ok || host == "" {
		return jsError(fmt.Errorf("host is required"))
	}

	withTrace := false
	withTraceVal := jsGetArg(args, 1)
	if withTraceVal.Type() != js.TypeUndefined {
		if withTraceVal.Type() != js.TypeBoolean {
			return jsError(fmt.Errorf("withTrace must be a boolean"))
		}
		withTrace = withTraceVal.Bool()
	}

	headers := make(map[string]string)
	headersJS := jsGetArg(args, 2)
	if headersJS.Type() == js.TypeObject {
		keys := js.Global().Get("Object").Call("keys", headersJS)
		length := keys.Get("length").Int()
		for i := 0; i < length; i++ {
			key := keys.Index(i).String()
			value := headersJS.Get(key)
			if value.Type() == js.TypeString {
				headers[key] = value.String()
			}
		}
	}

	return js.ValueOf(createHttpCaller(host, withTrace, headers))
}	
`)
}

func (g *WasmGenerator) generateServiceWasmBindings(sb *strings.Builder, svc *DeclService) {
	// Generate JS wrapper functions for each method
	for _, method := range svc.Methods {
		g.generateMethodWasmWrapper(sb, svc, method)
	}

	// Generate service object creator
	g.generateServiceObjectCreator(sb, svc)

	// Generate JS client creator for this service
	g.generateServiceClientFactory(sb, svc)
}

func (g *WasmGenerator) generateMethodWasmWrapper(sb *strings.Builder, svc *DeclService, method *DeclServiceMethod) {
	svcName := svc.Name.Name
	methodName := method.Name.Name
	funcName := fmt.Sprintf("js%s%s", svcName, methodName)
	numArgs := len(method.Args)

	sb.WriteString(fmt.Sprintf("func %s(serviceImpl %s, args []js.Value) any {\n", funcName, svcName))

	// Check if implementation is set
	sb.WriteString("\tif serviceImpl == nil {\n")
	sb.WriteString(fmt.Sprintf("\t\treturn jsPromiseRejected(fmt.Errorf(\"%s implementation not set\"))\n", svcName))
	sb.WriteString("\t}\n\n")

	// Parse options (last argument)
	sb.WriteString(fmt.Sprintf("\topts := createJsCallOptions(args, %d)\n\n", numArgs))

	// Check cache first
	sb.WriteString("\tif cached, found := opts.GetCache(); found {\n")
	if len(method.Returns) == 0 {
		sb.WriteString("\t\t_ = cached\n")
		sb.WriteString("\t\treturn js.Global().Get(\"Promise\").Call(\"resolve\", js.Undefined())\n")
	} else {
		sb.WriteString("\t\treturn js.Global().Get(\"Promise\").Call(\"resolve\", cached)\n")
	}
	sb.WriteString("\t}\n\n")

	// Return a promise
	sb.WriteString("\treturn jsPromise(func(resolve, reject js.Value) {\n")

	// Parse arguments
	for i, arg := range method.Args {
		g.generateArgParser(sb, arg, i)
	}

	// Create context with options (supports abort signal)
	sb.WriteString("\t\tctx, cleanup := opts.Context(context.Background())\n")
	sb.WriteString("\t\tdefer cleanup()\n\n")

	// Call the implementation
	sb.WriteString("\t\t")

	// Build return variable assignments
	var returnVars []string
	for _, ret := range method.Returns {
		returnVars = append(returnVars, toLowerFirst(ret.Name.Name))
	}
	returnVars = append(returnVars, "err")

	sb.WriteString(strings.Join(returnVars, ", "))
	sb.WriteString(" := ")
	sb.WriteString(fmt.Sprintf("serviceImpl.%s(ctx", methodName))

	// Add arguments
	for _, arg := range method.Args {
		sb.WriteString(", ")
		sb.WriteString(toLowerFirst(arg.Name.Name))
	}
	sb.WriteString(")\n")

	// Handle error
	sb.WriteString("\t\tif err != nil {\n")
	sb.WriteString("\t\t\treject.Invoke(jsError(err))\n")
	sb.WriteString("\t\t\treturn\n")
	sb.WriteString("\t\t}\n\n")

	// Build result and cache
	if len(method.Returns) == 0 {
		sb.WriteString("\t\tresolve.Invoke(js.Undefined())\n")
	} else if len(method.Returns) == 1 {
		ret := method.Returns[0]
		retVar := toLowerFirst(ret.Name.Name)
		sb.WriteString(fmt.Sprintf("\t\t_result := jsValueFromGo(%s)\n", retVar))
		sb.WriteString("\t\topts.SetCache(_result)\n")
		sb.WriteString("\t\tresolve.Invoke(_result)\n")
	} else {
		// Multiple returns - create an object
		sb.WriteString("\t\t_resultObj := js.Global().Get(\"Object\").New()\n")
		for _, ret := range method.Returns {
			retVar := toLowerFirst(ret.Name.Name)
			sb.WriteString(fmt.Sprintf("\t\t_resultObj.Set(\"%s\", jsValueFromGo(%s))\n", toCamelCase(ret.Name.Name), retVar))
		}
		sb.WriteString("\t\topts.SetCache(_resultObj)\n")
		sb.WriteString("\t\tresolve.Invoke(_resultObj)\n")
	}

	sb.WriteString("\t})\n")
	sb.WriteString("}\n\n")
}

func (g *WasmGenerator) generateArgParser(sb *strings.Builder, arg *DeclNameTypePair, index int) {
	argName := toLowerFirst(arg.Name.Name)
	argType := arg.Type

	switch t := argType.(type) {
	case *DeclStringType:
		sb.WriteString(fmt.Sprintf("\t\t%s, _ := jsGetStringArg(args, %d)\n", argName, index))
	case *DeclNumberType:
		sb.WriteString(fmt.Sprintf("\t\t%s := %s(jsGetArg(args, %d).Int())\n", argName, g.getGoNumberType(t.Name.Name), index))
	case *DeclBoolType:
		sb.WriteString(fmt.Sprintf("\t\t%s := jsGetArg(args, %d).Bool()\n", argName, index))
	case *DeclTimestampType:
		sb.WriteString(fmt.Sprintf("\t\t%sMs := jsGetArg(args, %d).Float()\n", argName, index))
		sb.WriteString(fmt.Sprintf("\t\t%s := time.UnixMilli(int64(%sMs))\n", argName, argName))
	case *DeclAnyType:
		sb.WriteString(fmt.Sprintf("\t\t%sJS := jsGetArg(args, %d)\n", argName, index))
		sb.WriteString(fmt.Sprintf("\t\tvar %s any\n", argName))
		sb.WriteString(fmt.Sprintf("\t\tif %sJS.Truthy() {\n", argName))
		sb.WriteString(fmt.Sprintf("\t\t\t%sJSON := js.Global().Get(\"JSON\").Call(\"stringify\", %sJS).String()\n", argName, argName))
		sb.WriteString(fmt.Sprintf("\t\t\tjson.Unmarshal([]byte(%sJSON), &%s)\n", argName, argName))
		sb.WriteString("\t\t}\n")
	case *DeclArrayType, *DeclMapType:
		sb.WriteString(fmt.Sprintf("\t\t%sJS := jsGetArg(args, %d)\n", argName, index))
		sb.WriteString(fmt.Sprintf("\t\tvar %s %s\n", argName, g.declTypeToGoTypeString(argType)))
		sb.WriteString(fmt.Sprintf("\t\tif %sJS.Truthy() {\n", argName))
		sb.WriteString(fmt.Sprintf("\t\t\t%sJSON := js.Global().Get(\"JSON\").Call(\"stringify\", %sJS).String()\n", argName, argName))
		sb.WriteString(fmt.Sprintf("\t\t\tjson.Unmarshal([]byte(%sJSON), &%s)\n", argName, argName))
		sb.WriteString("\t\t}\n")
	case *DeclCustomType:
		typeName := t.Name.Name
		if _, isEnum := g.enums[typeName]; isEnum {
			// Enum - treat as string
			sb.WriteString(fmt.Sprintf("\t\t%sStr, _ := jsGetStringArg(args, %d)\n", argName, index))
			sb.WriteString(fmt.Sprintf("\t\tvar %s %s\n", argName, typeName))
			sb.WriteString(fmt.Sprintf("\t\tjson.Unmarshal([]byte(\"\\\"\"+%sStr+\"\\\"\"), &%s)\n", argName, argName))
		} else {
			// Model - parse as JSON
			sb.WriteString(fmt.Sprintf("\t\t%sJS := jsGetArg(args, %d)\n", argName, index))
			sb.WriteString(fmt.Sprintf("\t\tvar %s *%s\n", argName, typeName))
			sb.WriteString(fmt.Sprintf("\t\tif %sJS.Truthy() {\n", argName))
			sb.WriteString(fmt.Sprintf("\t\t\t%sJSON := js.Global().Get(\"JSON\").Call(\"stringify\", %sJS).String()\n", argName, argName))
			sb.WriteString(fmt.Sprintf("\t\t\tjson.Unmarshal([]byte(%sJSON), &%s)\n", argName, argName))
			sb.WriteString("\t\t}\n")
		}
	default:
		sb.WriteString(fmt.Sprintf("\t\t%s := jsGetArg(args, %d) // unsupported type\n", argName, index))
	}
}

func (g *WasmGenerator) declTypeToGoTypeString(t DeclType) string {
	switch dt := t.(type) {
	case *DeclStringType:
		return "string"
	case *DeclNumberType:
		return g.getGoNumberType(dt.Name.Name)
	case *DeclBoolType:
		return "bool"
	case *DeclByteType:
		return "byte"
	case *DeclAnyType:
		return "any"
	case *DeclTimestampType:
		return "time.Time"
	case *DeclArrayType:
		return "[]" + g.declTypeToGoTypeString(dt.Type.(DeclType))
	case *DeclMapType:
		return "map[" + g.declTypeToGoTypeString(dt.KeyType.(DeclType)) + "]" + g.declTypeToGoTypeString(dt.ValueType.(DeclType))
	case *DeclCustomType:
		if _, isEnum := g.enums[dt.Name.Name]; isEnum {
			return dt.Name.Name
		}
		return "*" + dt.Name.Name
	default:
		return "any"
	}
}

func (g *WasmGenerator) getGoNumberType(name string) string {
	switch name {
	case "int8", "int16", "int32", "int64", "int":
		return name
	case "uint8", "uint16", "uint32", "uint64", "uint":
		return name
	case "float32", "float64":
		return name
	default:
		return "int"
	}
}

func (g *WasmGenerator) generateServiceObjectCreator(sb *strings.Builder, svc *DeclService) {
	svcName := svc.Name.Name
	funcName := fmt.Sprintf("create%sJSObject", svcName)

	sb.WriteString(fmt.Sprintf("func %s(serviceImpl %s) js.Value {\n", funcName, svcName))
	sb.WriteString("\tobj := js.Global().Get(\"Object\").New()\n")

	for _, method := range svc.Methods {
		methodNameCamel := toCamelCase(method.Name.Name)
		jsFuncName := fmt.Sprintf("js%s%s", svcName, method.Name.Name)
		sb.WriteString(fmt.Sprintf("\tobj.Set(\"%s\", js.FuncOf(func(_ js.Value, args []js.Value) any {\n", methodNameCamel))
		sb.WriteString(fmt.Sprintf("\t\treturn %s(serviceImpl, args)\n", jsFuncName))
		sb.WriteString("\t}))\n")
	}

	sb.WriteString("\treturn obj\n")
	sb.WriteString("}\n\n")
}

func (g *WasmGenerator) generateServiceClientFactory(sb *strings.Builder, svc *DeclService) {
	svcName := svc.Name.Name
	funcName := fmt.Sprintf("jsCreate%sClient", svcName)
	objCreator := fmt.Sprintf("create%sJSObject", svcName)

	sb.WriteString(fmt.Sprintf("func %s(_ js.Value, args []js.Value) any {\n", funcName))
	sb.WriteString("\thandlerIDVal := jsGetArg(args, 0)\n")
	sb.WriteString("\tif handlerIDVal.Type() != js.TypeNumber {\n")
	sb.WriteString("\t\treturn jsError(fmt.Errorf(\"handlerId must be a number\"))\n")
	sb.WriteString("\t}\n\n")
	sb.WriteString("\thandlerID := handlerIDVal.Int()\n")
	sb.WriteString("\tcaller, err := getCaller(handlerID)\n")
	sb.WriteString("\tif err != nil {\n")
	sb.WriteString("\t\treturn jsError(err)\n")
	sb.WriteString("\t}\n\n")
	sb.WriteString(fmt.Sprintf("\tserviceImpl := Create%sClient(caller)\n", svcName))
	sb.WriteString(fmt.Sprintf("\treturn %s(serviceImpl)\n", objCreator))
	sb.WriteString("}\n\n")
}

func (g *WasmGenerator) generateMainRegistration(sb *strings.Builder, services []*DeclService) {
	// Generate RegisterEllaServices function
	sb.WriteString("// RegisterEllaServices registers all services to the global 'ella' object\n")
	sb.WriteString("func RegisterEllaServices() {\n")
	sb.WriteString("\tobj := js.Global().Get(ellaObjName)\n")
	sb.WriteString("\tif obj.Type() != js.TypeObject {\n")
	sb.WriteString("\t\tobj = js.Global().Get(\"Object\").New()\n")
	sb.WriteString("\t}\n\n")

	// Add caller creation function first
	sb.WriteString("\tobj.Set(\"createHttpCaller\", js.FuncOf(jsCreateHttpCaller))\n")

	// Add cache invalidation functions
	sb.WriteString("\tobj.Set(\"invalidateCache\", js.FuncOf(jsInvalidateCache))\n")
	sb.WriteString("\tobj.Set(\"invalidateAllCache\", js.FuncOf(jsInvalidateAllCache))\n\n")

	for _, svc := range services {
		funcName := fmt.Sprintf("jsCreate%sClient", svc.Name.Name)
		sb.WriteString(fmt.Sprintf("\tobj.Set(\"create%sClient\", js.FuncOf(%s))\n", svc.Name.Name, funcName))
	}

	if g.allowExtensions {
		sb.WriteString("\n\tregisterEllaExtensions(obj)\n")
	}

	sb.WriteString("\n\tjs.Global().Set(ellaObjName, obj)\n")
	sb.WriteString("\tdispatchEllaReadyEvent()\n")
	sb.WriteString("}\n\n")

	// Generate dispatchEllaReadyEvent function
	sb.WriteString("func dispatchEllaReadyEvent() {\n")
	sb.WriteString("\tcustom := js.Global().Get(\"CustomEvent\")\n")
	sb.WriteString("\tif custom.Truthy() {\n")
	sb.WriteString("\t\tevt := custom.New(ellaEventReady)\n")
	sb.WriteString("\t\tjs.Global().Call(\"dispatchEvent\", evt)\n")
	sb.WriteString("\t\treturn\n")
	sb.WriteString("\t}\n\n")
	sb.WriteString("\tevtCtor := js.Global().Get(\"Event\")\n")
	sb.WriteString("\tif evtCtor.Truthy() {\n")
	sb.WriteString("\t\tevt := evtCtor.New(ellaEventReady)\n")
	sb.WriteString("\t\tjs.Global().Call(\"dispatchEvent\", evt)\n")
	sb.WriteString("\t}\n")
	sb.WriteString("}\n")
}

const wasmHelperFunctions = `
var (
	jsCache      = make(map[string]jsCacheEntry)
	jsCacheMutex sync.RWMutex

	handlersMap   = make(map[int]jsonrpc.Caller)
	handlersMutex sync.RWMutex
	nextHandlerID = 1
)

type jsCacheEntry struct {
	value     js.Value
	expiresAt time.Time
}

type jsCallOptions struct {
	signal   js.Value
	cacheKey string
	cacheTTL time.Duration
	timeout  time.Duration
}

func getNextHandlerID() int {
	handlersMutex.Lock()
	id := nextHandlerID
	nextHandlerID++
	handlersMutex.Unlock()
	return id
}

func getCaller(handlerID int) (jsonrpc.Caller, error) {
	handlersMutex.RLock()
	caller, found := handlersMap[handlerID]
	handlersMutex.RUnlock()
	if !found {
		return nil, fmt.Errorf("caller handler %d not found", handlerID)
	}
	return caller, nil
}

func (j *jsCallOptions) Context(parent context.Context) (context.Context, func()) {
	ctx, cancel := context.WithTimeout(parent, j.timeout)

	if !j.signal.Truthy() {
		return ctx, cancel
	}

	abortCallback := js.FuncOf(func(js.Value, []js.Value) any {
		cancel()
		return nil
	})

	j.signal.Call("addEventListener", "abort", abortCallback)

	cleanup := func() {
		j.signal.Call("removeEventListener", "abort", abortCallback)
		abortCallback.Release()
		cancel()
	}

	return ctx, cleanup
}

func (j *jsCallOptions) GetCache() (js.Value, bool) {
	if j.cacheKey == "" || j.cacheTTL <= 0 {
		return js.Undefined(), false
	}

	jsCacheMutex.RLock()
	entry, found := jsCache[j.cacheKey]
	jsCacheMutex.RUnlock()

	if !found {
		return js.Undefined(), false
	}

	if time.Now().After(entry.expiresAt) {
		jsCacheMutex.Lock()
		delete(jsCache, j.cacheKey)
		jsCacheMutex.Unlock()
		return js.Undefined(), false
	}

	return entry.value, true
}

func (j *jsCallOptions) SetCache(value js.Value) {
	if j.cacheKey == "" || j.cacheTTL <= 0 {
		return
	}

	jsCacheMutex.Lock()
	jsCache[j.cacheKey] = jsCacheEntry{
		value:     value,
		expiresAt: time.Now().Add(j.cacheTTL),
	}
	jsCacheMutex.Unlock()
}

func createJsCallOptions(args []js.Value, optIdx int) *jsCallOptions {
	opts := &jsCallOptions{
		timeout: 30 * time.Second,
	}

	optVal := jsGetArg(args, optIdx)
	if optVal.Type() != js.TypeObject {
		return opts
	}

	signalVal := optVal.Get("signal")
	if signalVal.Type() == js.TypeObject {
		opts.signal = signalVal
	}

	cacheKeyVal := optVal.Get("cacheKey")
	if cacheKeyVal.Type() == js.TypeString {
		opts.cacheKey = cacheKeyVal.String()
	}

	cacheTTLVal := optVal.Get("cacheTTL")
	if cacheTTLVal.Type() == js.TypeString {
		if dur, err := time.ParseDuration(cacheTTLVal.String()); err == nil {
			opts.cacheTTL = dur
		}
	}

	timeoutVal := optVal.Get("timeout")
	if timeoutVal.Type() == js.TypeString {
		if dur, err := time.ParseDuration(timeoutVal.String()); err == nil {
			opts.timeout = dur
		}
	}

	return opts
}

func jsInvalidateCache(_ js.Value, args []js.Value) any {
	key, ok := jsGetStringArg(args, 0)
	if !ok {
		return js.Undefined()
	}

	jsCacheMutex.Lock()
	delete(jsCache, key)
	jsCacheMutex.Unlock()

	return js.Undefined()
}

func jsInvalidateAllCache(_ js.Value, _ []js.Value) any {
	jsCacheMutex.Lock()
	jsCache = make(map[string]jsCacheEntry)
	jsCacheMutex.Unlock()

	return js.Undefined()
}

func jsPromise(executor func(resolve, reject js.Value)) js.Value {
	promiseCtor := js.Global().Get("Promise")
	executorFn := js.FuncOf(func(_ js.Value, args []js.Value) any {
		if len(args) < 2 {
			return nil
		}
		resolve := args[0]
		reject := args[1]
		go executor(resolve, reject)
		return nil
	})
	defer executorFn.Release()
	return promiseCtor.New(executorFn)
}

func jsError(err error) js.Value {
	jsError := js.Global().Get("Error")

	rpcErr, ok := err.(*jsonrpc.Error)
	if !ok {
		rpcErr = &jsonrpc.Error{
			Code:    jsonrpc.InternalError,
			Message: err.Error(),
			Cause:   nil,
		}
	}

	e := jsError.New(rpcErr.Message)
	e.Set("code", rpcErr.Code)
	if rpcErr.Cause != nil {
		e.Set("cause", rpcErr.Cause.Error())
	}

	return e
}

func jsPromiseRejected(err error) js.Value {
	return js.Global().Get("Promise").Call("reject", jsError(err))
}

func jsGetArg(args []js.Value, idx int) js.Value {
	if idx < len(args) {
		return args[idx]
	}
	return js.Undefined()
}

func jsGetStringArg(args []js.Value, idx int) (string, bool) {
	v := jsGetArg(args, idx)
	if v.Type() != js.TypeString {
		return "", false
	}
	return v.String(), true
}

func jsValueFromGo(v any) js.Value {
	if v == nil {
		return js.Null()
	}

	switch val := v.(type) {
	case bool:
		return js.ValueOf(val)
	case int, int8, int16, int32, int64:
		return js.ValueOf(val)
	case uint, uint8, uint16, uint32, uint64:
		return js.ValueOf(val)
	case float32, float64:
		return js.ValueOf(val)
	case string:
		return js.ValueOf(val)
	case time.Time:
		return js.ValueOf(val.UnixMilli())
	default:
		// For complex types, serialize to JSON and parse in JS
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			return js.Null()
		}
		return js.Global().Get("JSON").Call("parse", string(jsonBytes))
	}
}

`
