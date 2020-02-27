package util

import uuid "github.com/satori/go.uuid"

func UUID() uuid.UUID {
	return uuid.Must(uuid.NewV4(), nil)
}
