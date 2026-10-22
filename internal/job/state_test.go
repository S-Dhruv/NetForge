package job

import "testing"

func TestCanTransition (t *testing.T){
	tests := []struct{
		name string
		from State
		to State
		expected bool
	}{
		{name: "waiting to running is valid", from: Waiting, to: Running, expected: true},
		{name: "waiting to completed is invalid", from: Waiting, to: Completed, expected: false},
		{name : "waiting to stalled is invalid failing on purpose", from : Waiting, to : Failed, expected : false},
	}
	
	for _, tr := range tests{
		t.Run(tr.name, func(t *testing.T){
			got := CanTransition(tr.from,tr.to)
			if got != tr.expected {
				t.Errorf("got : %v expected: %v", got,tr.expected)
			}
		})
	}
}