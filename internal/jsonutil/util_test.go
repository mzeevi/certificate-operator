package jsonutil

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

type data struct {
	Key1 string `json:"key1"`
	Key2 string `json:"key2"`
}

func Test_ToJSON(t *testing.T) {
	type args struct {
		data interface{}
	}
	type want struct {
		result string
	}
	cases := map[string]struct {
		args args
		want want
	}{
		"ShouldReturnJsonString": {
			args: args{
				data: data{Key1: "data1", Key2: "data2"},
			},
			want: want{
				result: `{"key1":"data1","key2":"data2"}`,
			},
		},
		"ShouldReturnEmptyString": {
			args: args{
				data: map[data]string{
					{"key1", "key2"}: "Value1",
					{"key1", "key2"}: "Value2",
				},
			},
			want: want{
				result: "",
			},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := ToJSON(tc.args.data)
			if diff := cmp.Diff(tc.want.result, got); diff != "" {
				t.Errorf("ToJSON(...): -want result, +got result: %v", diff)
			}
		})
	}
}

func Test_IsJSONString(t *testing.T) {
	type args struct {
		data string
	}
	type want struct {
		result bool
	}
	cases := map[string]struct {
		args args
		want want
	}{
		"ShouldReturnTrue": {
			args: args{
				data: `{"key1":"data1","key2":"data2"}`,
			},
			want: want{
				result: true,
			},
		},
		"ShouldReturnFalse": {
			args: args{
				data: `{"key1", "key2"}: "Value1", "key1", "key2"}: "Value2",}`,
			},
			want: want{
				result: false,
			},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := IsJSONString(tc.args.data)
			if diff := cmp.Diff(tc.want.result, got); diff != "" {
				t.Errorf("IsJSONString(...): -want result, +got result: %v", diff)
			}
		})
	}
}
