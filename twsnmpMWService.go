package main

import "fmt"

type TwsnmpMWService struct{}

func (a *TwsnmpMWService) GetVersion() string {
	return fmt.Sprintf("%s(%s)", version, commit)
}
