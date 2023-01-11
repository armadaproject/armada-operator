package integration

import "os"

func CreateTempFile(data []byte) (*os.File, error) {
	f, err := os.CreateTemp("", "executor_test_")
	if err != nil {
		return nil, err
	}
	_, err = f.Write(data)
	if err != nil {
		return nil, err
	}
	return f, nil
}
