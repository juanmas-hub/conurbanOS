package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	globals "github.com/sisoputnfrba/tp-golang/globals/kernel"
	general "github.com/sisoputnfrba/tp-golang/kernel/utils/general"
)

func RecibirHandshakeIO(w http.ResponseWriter, r *http.Request) {

	decoder := json.NewDecoder(r.Body)
	var handshake globals.HandshakeIO
	err := decoder.Decode(&handshake)
	if err != nil {
		slog.Debug(fmt.Sprintf("Error al decodificar handshake: %s\n", err.Error()))
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar handshake"))
		return
	}

	globals.ListaIOsMutex.Lock()
	//slog.Debug(fmt.Sprintf("Se levantó una nueva instancia: %s, de IO: %s", handshake.NombreInstancia, handshake.NombreIO))
	agregarAInstanciasIOs(handshake)
	globals.ListaIOsMutex.Unlock()

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func RecibirHandshakeCPU(w http.ResponseWriter, r *http.Request) {

	//slog.Debug(fmt.Sprint("Me llego un handshaek de CPU"))

	decoder := json.NewDecoder(r.Body)
	var handshake globals.Handshake

	//slog.Debug(fmt.Sprint("Handshake: ", handshake))

	err := decoder.Decode(&handshake)
	if err != nil {
		slog.Debug(fmt.Sprintf("Error al decodificar handshake: %s\n", err.Error()))
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar handshake"))
		return
	}

	globals.ListaCPUsMutex.Lock()
	//slog.Debug(fmt.Sprintf("Se levantó una nueva CPU: %s, IP: %s, Puerto: %d", handshake.Nombre, handshake.IP, handshake.Puerto))
	agregarAListaCPUs(handshake)
	globals.ListaCPUsMutex.Unlock()

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func agregarAInstanciasIOs(handshake globals.HandshakeIO) {
	elementoAAgregar := globals.InstanciaIO{
		Handshake:        handshake,
		PidProcesoActual: -1,
	}
	io, existe := globals.MapaIOs[handshake.NombreIO]
	if !existe {
		io = globals.EntradaMapaIO{}
	}
	io.Instancias = append(io.Instancias, elementoAAgregar)
	globals.MapaIOs[handshake.NombreIO] = io

	//IntentarPasarAIO(handshake.NombreIO, elementoAAgregar)
}

func agregarAListaCPUs(handshake globals.Handshake) {
	elementoAAgregar := globals.ListaCpu{
		Handshake: handshake,
		EstaLibre: true,
	}
	globals.ListaCPUs = append(globals.ListaCPUs, elementoAAgregar)

	switch globals.KernelConfig.Scheduler_algorithm {
	case "FIFO", "SJF":
		general.Signal(globals.Sem_Cpus)
	case "SRT":
		//slog.Debug(fmt.Sprintf("Notificando replanificación en agregarAListaCPUs - Nueva CPU Libre"))
		//general.NotificarReplanifSRT()
	}
}
