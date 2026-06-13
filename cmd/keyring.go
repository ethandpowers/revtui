package main

import (
	"github.com/zalando/go-keyring"
)

const ServiceName = "revtui"

func SavePassword(host string, username string, password string) error {
	return keyring.Set(ServiceName, host+":"+username, password)
}

func GetPassword(host string, username string) (string, error) {
	return keyring.Get(ServiceName, host+":"+username)
}

func DeletePasswordFor(host string, username string) error {
	return keyring.Delete(ServiceName, host+":"+username)
}

func DeleteAllPasswords() error {
	return keyring.DeleteAll(ServiceName)
}
