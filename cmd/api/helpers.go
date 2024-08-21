package main

import (
	"log"
)

func background(fn func()) {
	go func() {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("%v", err)
			}
		}()

		fn()
	}()
}
