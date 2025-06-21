package utils_syscallController

import (
	"encoding/json"
	"log"
	"net/http"

	globals "github.com/sisoputnfrba/tp-golang/globals/kernel"
)

// Cuando la CPU detecta una syscall, nos envía a kernel y nosotros la manejamos:
// En todas las syscalls la CPU "se libera" y queda esperando para simular el tiempo que ejecuta el SO
// - En IO el proceso se bloquea, entonces directamente el planificador de corto plazo replanifica.
// - En INIT PROC la CPU no la indicamos como "libre" porque tiene que volver a ejecutar el mismo proceso

func RecibirIO(w http.ResponseWriter, r *http.Request) {
	// Recibo desde CPU la syscall IO y le envío solicitud a la IO correspondiente

	decoder := json.NewDecoder(r.Body)
	var syscallIO globals.SyscallIO
	err := decoder.Decode(&syscallIO)
	if err != nil {
		log.Printf("Error al decodificar syscallIO: %s\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar syscallIO"))
		return
	}

	go manejarIO(syscallIO)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func RecibirINIT_PROC(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var syscallINIT globals.SyscallInit
	err := decoder.Decode(&syscallINIT)
	if err != nil {
		log.Printf("Error al decodificar SyscallInit: %s\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar syscallINIT"))
		return
	}

	go manejarInit_Proc(syscallINIT)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func RecibirDUMP_MEMORY(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var syscallDUMP globals.SyscallDump
	err := decoder.Decode(&syscallDUMP)
	if err != nil {
		log.Printf("Error al decodificar SyscallDump: %s\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar SyscallDump"))
		return
	}

	go manejarDUMP_MEMORY(syscallDUMP)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func RecibirEXIT(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var syscallEXIT globals.SyscallExit
	err := decoder.Decode(&syscallEXIT)
	if err != nil {
		log.Printf("Error al decodificar SyscallExit: %s\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar SyscallExit"))
		return
	}

	go manejarEXIT(syscallEXIT)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}
