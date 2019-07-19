package common

import uuid "github.com/satori/go.uuid"

func SequentialUUID() string {
	return uuid.Must(uuid.NewV1(), nil).String()
}

func RandomUUID() string {
	return uuid.Must(uuid.NewV4(), nil).String()
}
