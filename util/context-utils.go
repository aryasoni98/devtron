/*
 * Copyright (c) 2024. Devtron Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package util

import (
	"context"
	"fmt"
	"reflect"
)

const (
	IsSuperAdminFlag = "isSuperAdmin"
	UserId           = "userId"
)

func SetSuperAdminInContext(ctx context.Context, isSuperAdmin bool) context.Context {
	ctx = context.WithValue(ctx, IsSuperAdminFlag, isSuperAdmin)
	return ctx
}

func GetIsSuperAdminFromContext(ctx context.Context) (bool, error) {
	flag := ctx.Value(IsSuperAdminFlag)

	if flag != nil && reflect.TypeOf(flag).Kind() == reflect.Bool {
		return flag.(bool), nil
	}
	return false, fmt.Errorf("context not valid, isSuperAdmin flag not set correctly %v", flag)
}

// SetTokenInContext - Set token in context
// NOTE: In OSS we don't have the token embedded in ctx already.
// TODO: Support NewRequestCtx in OSS as well.
func SetTokenInContext(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, "token", token)
}
