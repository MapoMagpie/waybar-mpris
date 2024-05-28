package main

import "testing"

func Test_cutStringByWidthAndStep(t *testing.T) {
	type args struct {
		str   string
		step  int
		width int
	}
	tests := []struct {
		name string
		want string
		args args
	}{
		{
			name: "cut title 1",
			args: args{
				str:   "Hello World",
				step:  0,
				width: 10,
			},
			want: "Hello Worl",
		},
		{
			name: "cut title 2",
			args: args{
				str:   "Hello World",
				step:  3,
				width: 12,
			},
			want: "Hello World",
		},
		{
			name: "cut title 3",
			args: args{
				str:   "Hello World",
				step:  6,
				width: 10,
			},
			want: "World > He",
		},
		{
			name: "cut title 4",
			args: args{
				str:   "Hello World",
				step:  11,
				width: 10,
			},
			want: "Hello Worl",
		},
		{
			name: "cut title 5",
			args: args{
				str:   "Hello World",
				step:  3,
				width: 10,
			},
			want: "lo World >",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cutStringByWidthAndStep(tt.args.str, tt.args.width, tt.args.step); got != tt.want {
				t.Errorf("cutStringByWidthAndStep() = %v, want %v", got, tt.want)
			}
		})
	}
}
