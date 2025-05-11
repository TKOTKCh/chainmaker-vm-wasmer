/*
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 *
 */

package types

import "testing"

func TestEngineType_LowerString(t *testing.T) {
	tests := []struct {
		name string
		t    EngineType
		want string
	}{
		// TODO: Add test cases.
		{
			name: "test1",
			t:    EngineType(5),
			want: "badgerdb",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.t.LowerString(); got != tt.want {
				t.Errorf("LowerString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEngineType_String(t *testing.T) {
	tests := []struct {
		name string
		t    EngineType
		want string
	}{
		// TODO: Add test cases.
		{
			name: "UnknownDb",
			t:    EngineType(0),
			want: "UnknownDb",
		},
		{
			name: "LevelDb",
			t:    EngineType(1),
			want: "LevelDb",
		},
		{
			name: "TikvDb",
			t:    EngineType(2),
			want: "TikvDb",
		},
		{
			name: "BadgerDb",
			t:    EngineType(5),
			want: "BadgerDb",
		},
		{
			name: "MySQL",
			t:    EngineType(3),
			want: "MySQL",
		},
		{
			name: "Sqlite",
			t:    EngineType(4),
			want: "Sqlite",
		},
		{
			name: "default",
			t:    EngineType(6),
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.t.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}
