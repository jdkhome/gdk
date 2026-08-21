// Copyright 2018 The Wire Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package di

import (
	"go/token"
)

// errorCollector 管理一组错误。零值表示空列表。
type errorCollector struct {
	errors []error
}

// add 将任意非 nil 错误追加到收集器中。
func (ec *errorCollector) add(errs ...error) {
	for _, e := range errs {
		if e != nil {
			ec.errors = append(ec.errors, e)
		}
	}
}

// mapErrors 返回一个新切片，用给定函数包装所有错误。
func mapErrors(errs []error, f func(error) error) []error {
	if len(errs) == 0 {
		return nil
	}
	newErrs := make([]error, len(errs))
	for i := range errs {
		newErrs[i] = f(errs[i])
	}
	return newErrs
}

// diErr 是带可选位置的错误。
type diErr struct {
	error    error
	position token.Position
}

// notePosition 在错误尚无位置信息时为其附加位置信息
//（如果已有则不附加）。
//
// notePosition 通常在错误沿调用栈向上传播时被多次调用，
// 因此对已有 *diErr 调用 notePosition 不会修改其位置，
// 因为假设更深的调用拥有更精确的位置信息，
// 即错误的来源位置。
func notePosition(p token.Position, e error) error {
	switch e.(type) {
	case nil:
		return nil
	case *diErr:
		return e
	default:
		return &diErr{error: e, position: p}
	}
}

// notePositionAll 用给定位置包装一组错误。
func notePositionAll(p token.Position, errs []error) []error {
	return mapErrors(errs, func(e error) error {
		return notePosition(p, e)
	})
}

// Error 返回错误信息；若位置有效则在前面加上位置信息。
func (w *diErr) Error() string {
	if !w.position.IsValid() {
		return w.error.Error()
	}
	return w.position.String() + ": " + w.error.Error()
}
