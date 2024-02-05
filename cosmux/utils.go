package main

import "fmt"

var uniqueId int = 0

func UniqueID() string {
	uniqueId++
	return fmt.Sprint(uniqueId)
}
