package resp

import (
	"bufio"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestResp_readLine(t *testing.T) {
	tests := []struct {
		name     string
		reader   *bufio.Reader
		wantLine []byte
		wantN    int
		wantErr  bool
	}{
		{
			name:     "1",
			reader:   bufio.NewReader(strings.NewReader("1")),
			wantLine: nil,
			wantN:    0,
			wantErr:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Resp{
				reader: tt.reader,
			}
			gotLine, gotN, err := r.readLine()
			if (err != nil) != tt.wantErr {
				t.Errorf("readLine() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			fmt.Println("gotLine", string(gotLine))
			if !reflect.DeepEqual(gotLine, tt.wantLine) {
				t.Errorf("readLine() gotLine = %v, want %v", gotLine, tt.wantLine)
			}
			if gotN != tt.wantN {
				t.Errorf("readLine() gotN = %v, want %v", gotN, tt.wantN)
			}
		})
	}
}
