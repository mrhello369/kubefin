/*
Copyright 2023 The KubeFin Authors

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

package defaultcloud

import (
	"fmt"
	"reflect"
	"testing"
)

func Test_parseGPUDevicePriceList(t *testing.T) {
	tests := []struct {
		name      string
		priceList string
		wantMap   map[string]float64
		wantErr   error
	}{
		{
			name:      "empty",
			priceList: "",
			wantMap:   nil,
			wantErr:   fmt.Errorf("failed to parse GPU device price list"),
		},
		{
			name:      "correct input",
			priceList: "T1:12;A100:15",
			wantMap: map[string]float64{
				"A100": 15,
				"T1":   12,
			},
			wantErr: nil,
		},
		{
			name:      "wrong input",
			priceList: "T1:12:12;A100:15",
			wantMap:   nil,
			wantErr:   fmt.Errorf("failed to parse GPU device price list"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ret, err := parseGPUDevicePriceList(tt.priceList)
			if !reflect.DeepEqual(ret, tt.wantMap) || !reflect.DeepEqual(err, tt.wantErr) {
				t.Errorf("parseGPUDevicePriceList() want (%v, %v), but got (%v, %v)", tt.wantMap, tt.wantErr,
					ret, err)
			}
		})
	}
}
