package websocket

import "testing"

func TestParseCommand(t *testing.T) {
	tests := []struct {
		raw  string
		want command
		err  error
	}{
		{`{"type":"join","room_id":1}`, joinCmd{room: 1}, nil},
		{`{"type":"leave","room_id":2}`, leaveCmd{room: 2}, nil},
		{`{"type":"chat","room_id":1,"content":"hi"}`, roomSendCmd{room: 1, content: "hi"}, nil},
		{`{"type":"media","room_id":1,"media_id":9}`, roomSendCmd{room: 1, mediaID: 9}, nil},
		{`{"type":"dm","to":5,"content":"hey"}`, dmCmd{to: 5, content: "hey"}, nil},
		{`{"type":"block","user_id":5}`, blockCmd{user: 5}, nil},
		{`{"type":"unblock","user_id":5}`, unblockCmd{user: 5}, nil},
		{`{"type":"chat","room_id":1,"content":"   "}`, nil, errEmpty},
		{`{"type":"dm","content":"hey"}`, nil, errTarget},
		{`{"type":"dm","to":5}`, nil, errEmpty},
		{`{"type":"join"}`, nil, errRoom},
		{`{"type":"block"}`, nil, errTarget},
		{`{"type":"dance","room_id":1}`, nil, errUnknown},
		{`not json`, nil, errFrame},
	}
	for _, tt := range tests {
		got, err := parseCommand([]byte(tt.raw))
		if got != tt.want || err != tt.err {
			t.Errorf("parseCommand(%s) = (%#v, %v), want (%#v, %v)", tt.raw, got, err, tt.want, tt.err)
		}
	}
}
