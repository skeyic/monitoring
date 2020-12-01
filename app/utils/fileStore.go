package utils

import "io/ioutil"

func SaveToFile(fileName string, content []byte) (err error) {
	err = ioutil.WriteFile(fileName, content, 0666)
	return
}

func ReadFromFile(fileName string) (content []byte, err error) {
	return ioutil.ReadFile(fileName)
}
