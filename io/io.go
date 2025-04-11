package main

import (
	"log"

	globals "github.com/sisoputnfrba/tp-golang/globals/io"
	utils_io "github.com/sisoputnfrba/tp-golang/io/utils"
	utils_logger "github.com/sisoputnfrba/tp-golang/utils/loggers"
)

func main() {
	utils_logger.ConfigurarLogger("io.log")

	globals.IoConfig = utils_io.IniciarConfiguracion("config.json")
	if globals.IoConfig == nil {
		log.Fatal("No se pudo iniciar el config")
	}

}
