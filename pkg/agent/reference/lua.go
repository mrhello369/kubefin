/*
Copyright 2022 The Karmada Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package reference

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	lua "github.com/yuin/gopher-lua"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	luajson "layeh.com/gopher-json"
)

func ExtractPodLabelSelector(obj *unstructured.Unstructured, script string) (map[string]string, error) {
	vm := newLuaVM(false)
	results, err := vm.runScript(script, "ExtractPodLabelSelector", 1, obj)
	if err != nil {
		return nil, err
	}

	labelSelector := map[string]string{}
	luaResult := results[0]
	if t, ok := luaResult.(*lua.LTable); ok {
		t.ForEach(func(key lua.LValue, value lua.LValue) {
			labelSelector[key.String()] = value.String()
		})
		return labelSelector, nil
	}

	return nil, fmt.Errorf("execute lua script unexpected result")
}

// Following code is referring to: https://github.com/karmada-io/karmada/blob/master/pkg/resourceinterpreter/customized/declarative/luavm/lua.go
// luaVM Defines a struct that implements the luaVM.
type luaVM struct {
	// UseOpenLibs flag to enable open libraries. Libraries are disabled by default while running, but enabled during testing to allow the use of print statements.
	UseOpenLibs bool
}

func newLuaVM(useOpenLibs bool) *luaVM {
	vm := &luaVM{
		UseOpenLibs: useOpenLibs,
	}
	return vm
}

// NewLuaState creates a new lua state.
func (vm *luaVM) newLuaState() (*lua.LState, error) {
	l := lua.NewState(lua.Options{
		SkipOpenLibs: !vm.UseOpenLibs,
	})
	// Opens table library to allow access to functions to manipulate tables
	err := vm.setLib(l)
	if err != nil {
		return nil, err
	}
	return l, err
}

// RunScript got a lua vm from pool, and execute script with given arguments.
func (vm *luaVM) runScript(script string, fnName string, nRets int, args ...interface{}) ([]lua.LValue, error) {
	l, err := vm.newLuaState()
	if err != nil {
		return nil, err
	}
	l.Pop(l.GetTop())

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	l.SetContext(ctx)

	err = l.DoString(script)
	if err != nil {
		return nil, err
	}

	vArgs := make([]lua.LValue, len(args))
	for i, arg := range args {
		vArgs[i], err = decodeValue(l, arg)
		if err != nil {
			return nil, err
		}
	}

	f := l.GetGlobal(fnName)
	if f.Type() == lua.LTNil {
		return nil, fmt.Errorf("not found function %v", fnName)
	}
	if f.Type() != lua.LTFunction {
		return nil, fmt.Errorf("%s is not a function: %s", fnName, f.Type())
	}

	err = l.CallByParam(lua.P{
		Fn:      f,
		NRet:    nRets,
		Protect: true,
	}, vArgs...)
	if err != nil {
		return nil, err
	}

	// get rets from stack: [ret1, ret2, ret3 ...]
	rets := make([]lua.LValue, nRets)
	for i := range rets {
		rets[i] = l.Get(i + 1)
	}
	// pop all the values in stack
	l.Pop(l.GetTop())
	return rets, nil
}

func (vm *luaVM) setLib(l *lua.LState) error {
	for _, pair := range []struct {
		n string
		f lua.LGFunction
	}{
		{lua.LoadLibName, lua.OpenPackage},
		{lua.BaseLibName, lua.OpenBase},
		{lua.TabLibName, lua.OpenTable},
		{lua.StringLibName, lua.OpenString},
		{lua.MathLibName, lua.OpenMath},
	} {
		if err := l.CallByParam(lua.P{
			Fn:      l.NewFunction(pair.f),
			NRet:    0,
			Protect: true,
		}, lua.LString(pair.n)); err != nil {
			return err
		}
	}
	return nil
}

// nolint:gocyclo
func decodeValue(L *lua.LState, value interface{}) (lua.LValue, error) {
	// We handle simple type without json for better performance.
	switch converted := value.(type) {
	case []interface{}:
		arr := L.CreateTable(len(converted), 0)
		for _, item := range converted {
			v, err := decodeValue(L, item)
			if err != nil {
				return nil, err
			}
			arr.Append(v)
		}
		return arr, nil
	case map[string]interface{}:
		tbl := L.CreateTable(0, len(converted))
		for key, item := range converted {
			v, err := decodeValue(L, item)
			if err != nil {
				return nil, err
			}
			tbl.RawSetString(key, v)
		}
		return tbl, nil
	case nil:
		return lua.LNil, nil
	}

	v := reflect.ValueOf(value)
	switch {
	case v.CanInt():
		return lua.LNumber(v.Int()), nil
	case v.CanUint():
		return lua.LNumber(v.Uint()), nil
	case v.CanFloat():
		return lua.LNumber(v.Float()), nil
	}

	switch t := v.Type(); t.Kind() {
	case reflect.String:
		return lua.LString(v.String()), nil
	case reflect.Bool:
		return lua.LBool(v.Bool()), nil
	case reflect.Pointer:
		if v.IsNil() {
			return lua.LNil, nil
		}
	}

	// Other types can't be handled, ask for help from json
	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("json Marshal obj %#v error: %v", value, err)
	}

	lv, err := luajson.Decode(L, data)
	if err != nil {
		return nil, fmt.Errorf("lua Decode obj %#v error: %v", value, err)
	}
	return lv, nil
}
