package main

import (
	"fmt"

	"github.com/bytearena/core/common/types/datastructures"
)

func dump(list datastructures.DLL, title string) {

	fmt.Println("----- FORWARD -----", title)

	node := list.Head
	for node != nil {
		fmt.Println(node.Val)
		node = node.Next
	}

	fmt.Println("")
	fmt.Println("")
}

func dumpReverse(list datastructures.DLL, title string) {

	fmt.Println("----- REVERSE -----", title)

	node := list.Tail
	for node != nil {
		fmt.Println(node.Val)
		node = node.Prev
	}

	fmt.Println("")
	fmt.Println("")
}

func main() {
	list := datastructures.DLL{}
	list.Append(0)
	list.Append(1)
	list.Append(2)
	list.Append(3)
	list.Append(4)

	dump(list, "Full list")
	dumpReverse(list, "Full list")

	list.RemoveVal(2)
	dump(list, "Remove(2)")
	dumpReverse(list, "Remove(2)")

	list.RemoveVal(0)
	dump(list, "Remove(0)")
	dumpReverse(list, "Remove(0)")

	list.RemoveVal(0)
	dump(list, "Remove(0) again")
	dumpReverse(list, "Remove(0) again")

	list.RemoveVal(4)
	dump(list, "Remove(4)")
	dumpReverse(list, "Remove(4)")

	list.InsertBefore(list.Head.Next, "inserted")
	dump(list, "InsertBefore(3)")
	dumpReverse(list, "InsertBefore(3)")

	list.InsertBefore(list.Head, "insertedHead")
	dump(list, "InsertBefore(head)")
	dumpReverse(list, "InsertBefore(head)")

	fmt.Println("ISEMPTY", list.Empty())

	list.Clear()
	dump(list, "CLEAR")
	dumpReverse(list, "CLEAR")

	fmt.Println("ISEMPTY", list.Empty())
}
