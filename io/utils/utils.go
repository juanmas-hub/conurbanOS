package utils

import (
	"encoding/json"
	"log"
	"os"

	globals "github.com/sisoputnfrba/tp-golang/globals/io"
)

func IniciarConfiguracion(filePath string) *globals.Io_Config {
	var config *globals.Io_Config
	configFile, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer configFile.Close()

	jsonParser := json.NewDecoder(configFile)
	jsonParser.Decode(&config)

	return config
}
