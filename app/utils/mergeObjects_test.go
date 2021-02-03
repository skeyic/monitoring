package utils

import (
	"fmt"
	"testing"
)

type Student struct {
	id int64
}

func (s Student) ID() int64 {
	return s.id
}

func TestMergeDescendObjects(t *testing.T) {
	var (
		s1 = []ToMergeObject{
			Student{
				id: 20,
			},
			Student{
				id: 19,
			},
			Student{
				id: 15,
			},
			Student{
				id: 13,
			},
			Student{
				id: 9,
			},
		}
		s2 = []ToMergeObject{
			Student{
				id: 13,
			},
			Student{
				id: 4,
			},
			Student{
				id: 3,
			},
			Student{
				id: 2,
			},
		}
	)

	sl := MergeDescendObjects(s1, s2)
	for _, s := range sl {
		fmt.Print(s.ID(), ",")
	}
	fmt.Println()
}

func TestArray(t *testing.T) {
	var (
		sl = []string{"a", "b", "c", "d"}
		lb = len(sl)
	)

	sl = append([]string{"e"}, sl...)
	la := len(sl)
	fmt.Println(sl[:la-lb])

	lb = len(sl)
	sl = append([]string{"f"}, sl...)
	la = len(sl)
	fmt.Println(sl[:la-lb])

	lb = len(sl)
	la = len(sl)
	fmt.Println(sl[:la-lb])
}
