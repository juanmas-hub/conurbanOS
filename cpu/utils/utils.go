package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io" //NUEVO Necesario para io.ReadAll
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	globals "github.com/sisoputnfrba/tp-golang/globals/cpu"
	globals_cpu "github.com/sisoputnfrba/tp-golang/globals/cpu"
)

func IniciarConfiguracion(filePath string) *globals.Cpu_Config {
	var config *globals.Cpu_Config
	configFile, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer configFile.Close()

	jsonParser := json.NewDecoder(configFile)
	jsonParser.Decode(&config)

	return config
}

func EnviarMensaje(ip string, puerto int64, mensajeTxt string) {
	mensaje := globals.Mensaje{Mensaje: mensajeTxt}
	body, err := json.Marshal(mensaje)
	if err != nil {
		log.Printf("error codificando mensaje: %s", err.Error())
	}

	// Posible problema con el int64 del puerto
	url := fmt.Sprintf("http://%s:%d/mensajeDeCpu", ip, puerto)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("error enviando mensaje a ip:%s puerto:%d", ip, puerto)
	}

	log.Printf("respuesta del servidor: %s", resp.Status)
}

func RecibirMensajeDeKernel(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var mensaje globals.Mensaje
	err := decoder.Decode(&mensaje)
	if err != nil {
		log.Printf("Error al decodificar mensaje: %s\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar mensaje"))
		return
	}

	log.Println("Me llego un mensaje de Kernel")
	log.Printf("%+v\n", mensaje)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func HandshakeAKernel(ip string, puerto int64, nombreCPU string, ipCPU string, puertoCPU int64) {

	handshake := globals.HandshakeCPU{
		Nombre: nombreCPU,
		IP:     ipCPU,
		Puerto: puertoCPU,
	}
	body, err := json.Marshal(handshake)
	if err != nil {
		log.Printf("error codificando mensaje: %s", err.Error())
	}

	url := fmt.Sprintf("http://%s:%d/handshakeCPU", ip, puerto)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))

	if err != nil {
		log.Printf("error en el handshake a ip:%s puerto:%d", ip, puerto)
	}

	log.Printf("respuesta del servidor (handshake): %s", resp.Status)

}

var ColaDeEjecucion = make(chan globals.PCB, 10)

func RecibirPCBDeKernel(w http.ResponseWriter, r *http.Request) {
	var pcb globals.PCB
	err := json.NewDecoder(r.Body).Decode(&pcb)
	if err != nil {
		log.Printf("Error al decodificar PCB: %s\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar PCB"))
		return
	}

	log.Printf("PCB recibido: PID=%d PC=%d\n", pcb.Pid, pcb.PC)

	ColaDeEjecucion <- pcb

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func EnviarSolicitudInstruccion(pid int64, pc int64) (string, error) {
	solicitud := globals.SolicitudInstruccion{
		Pid: pid,
		PC:  pc,
	}

	body, err := json.Marshal(solicitud)
	if err != nil {
		return "", fmt.Errorf("error codificando solicitud a Memoria: %w", err)
	}

	url := fmt.Sprintf("http://%s:%d/obtenerInstruccion", globals_cpu.CpuConfig.Ip_memory, globals_cpu.CpuConfig.Port_memory)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return "", fmt.Errorf("error haciendo POST a Memoria: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("memoria respondió con error: %d", resp.StatusCode)
	}

	var respuesta struct {
		Instruccion string `json:"instruccion"`
	}
	err = json.NewDecoder(resp.Body).Decode(&respuesta)
	if err != nil {
		return "", fmt.Errorf("error al decodificar respuesta de Memoria: %w", err)
	}

	log.Printf("Instrucción recibida de Memoria: %s", respuesta.Instruccion)
	return respuesta.Instruccion, nil
}

func Decode(instruccion string) (globals.InstruccionDecodificada, error) {
	partes := strings.SplitN(instruccion, " ", 2) //divide la instruccion de los parametros
	nombre := partes[0]
	parametrosStr := ""
	if len(partes) > 1 {
		parametrosStr = partes[1]
		parametrosStr = strings.Trim(parametrosStr, "()") //borra los parentesis que deja la funcion anterior
	}
	parametros := strings.Split(parametrosStr, ", ") //divide los argumentos y los deja separados en un array de strings

	instDeco := globals.InstruccionDecodificada{
		Nombre:     nombre,
		Parametros: parametros,
	}

	switch nombre { //para cada instruccion devuelve error si se le pasa una cantidad incorrecta de parametros
	case "WRITE":
		if len(parametros) != 2 {
			return globals.InstruccionDecodificada{}, fmt.Errorf("formato incorrecto para WRITE: se esperan 2 parámetros (dirección, datos)")
		}

	case "READ":
		if len(parametros) != 2 {
			return globals.InstruccionDecodificada{}, fmt.Errorf("formato incorrecto para READ: se esperan 2 parámetros (dirección, tamaño)")
		}

	case "GOTO":
		if len(parametros) != 1 {
			return globals.InstruccionDecodificada{}, fmt.Errorf("formato incorrecto para GOTO: se espera 1 parámetro (valor)")
		}

	case "IO":
		if len(parametros) != 2 {
			return globals.InstruccionDecodificada{}, fmt.Errorf("formato incorrecto para IO: se esperan 2 parámetros (dispositivo, tiempo)")
		}

	case "INIT_PROC":
		if len(parametros) != 2 {
			return globals.InstruccionDecodificada{}, fmt.Errorf("formato incorrecto para INIT_PROC: se esperan 2 parámetros (archivo, tamaño)")
		}

	case "DUMP_MEMORY":
		if len(parametros) != 0 {
			return globals.InstruccionDecodificada{}, fmt.Errorf("formato incorrecto para DUMP_MEMORY: no se esperan parámetros")
		}
	case "EXIT":
		if len(parametros) != 0 {
			return globals.InstruccionDecodificada{}, fmt.Errorf("formato incorrecto para EXIT: no se esperan parámetros")
		}
	case "NOOP":
		if len(parametros) != 0 {
			return globals.InstruccionDecodificada{}, fmt.Errorf("formato incorrecto para NOOP: no se esperan parámetros")
		}
	default:
		return globals.InstruccionDecodificada{}, fmt.Errorf("instrucción desconocida: %s", nombre)
	}

	return instDeco, nil
}

//Test para enviar a memoria ↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓

// EnviarDireccionAMemoria simula el envío de una dirección física (como string por ahora) a la Memoria.
// FUTURO: Esta función se adaptará para recibir un int64 como dirección física real.
func EnviarDireccionAMemoria(pid int64, physicalAddressStr string) error { // FUTURO: physicalAddressStr será physicalAddress int64
	// Construir el payload JSON
	// FUTURO: El payload podría contener más campos relevantes para la operación de escritura/lectura (ej. datos a escribir, tamaño a leer).
	payload := map[string]interface{}{
		"pid":              pid,
		"direccion_fisica": physicalAddressStr, // FUTURO: Aquí se usaría physicalAddress (int64)
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error codificando payload JSON para Memoria: %s", err.Error())
		return fmt.Errorf("error codificando payload JSON para Memoria: %w", err)
	}

	// Construir la URL del endpoint de Memoria
	// Asumiendo que el endpoint para recibir una dirección física es "/recibirDireccionFisica"
	// FUTURO: Los endpoints para WRITE y READ serán específicos, como "/escribirMemoria" o "/leerMemoria". [2]
	url := fmt.Sprintf("http://%s:%d/recibirDireccionFisica", globals_cpu.CpuConfig.Ip_memory, globals_cpu.CpuConfig.Port_memory)

	// Crear un cliente HTTP con un timeout.
	// En el código actual del usuario, no hay un http.Client pre-inicializado y reutilizable.
	// Por lo tanto, se crea uno nuevo aquí.
	// FUTURO: Cuando se refactorice la CPU para tener un http.Client persistente (ej. en globals_cpu.GlobalCPU.MemoryClient),
	// se debería usar ese cliente en lugar de crear uno nuevo aquí.
	client := &http.Client{Timeout: 5 * time.Second} // Añadir un timeout para evitar bloqueos

	// Enviar la solicitud HTTP POST
	resp, err := client.Post(url, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		log.Printf("Error enviando solicitud a Memoria (%s): %s", url, err.Error())
		return fmt.Errorf("error enviando solicitud a Memoria (%s): %w", url, err)
	}
	defer resp.Body.Close() // Asegurarse de cerrar el cuerpo de la respuesta

	// Verificar la respuesta de Memoria
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body) // Leer el cuerpo para el log de error
		log.Printf("Memoria respondió con error %d: %s", resp.StatusCode, string(bodyBytes))
		return fmt.Errorf("memoria respondió con error %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Log obligatorio (adaptado para este ejemplo) [2]
	// FUTURO: El log se adaptará al formato específico de "Lectura/Escritura Memoria" del enunciado. [2]
	log.Printf("PID: %d - Dirección enviada a Memoria: %s", pid, physicalAddressStr) // FUTURO: Aquí se usaría physicalAddress (int64)

	return nil
}

//Test para enviar a memoria ↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑

type ResultadoEjecucion int

const (
	CONTINUAR_EJECUCION ResultadoEjecucion = iota
	PONERSE_ESPERA
	ERROR_EJECUCION
)

func Execute(instDeco globals.InstruccionDecodificada, pcb globals.PCB) (ResultadoEjecucion, error) {
	switch instDeco.Nombre { //En cada caso habria que extraer los parametros del string y pasarlos a una variable de su tipo de dato, luego ejecutar la logica correspondiente
	// Tambien hay que actualizar el PC, hacerle ++ o actualizarlo al valor del GOTO
	case "NOOP":
		time.Sleep(1 * time.Second)
		pcb.PC++
		return CONTINUAR_EJECUCION, nil
	case "WRITE":
		//pasar direccion logica por mmu y mandarle la fisica a memoria junto el dato a escribir
		pcb.PC++
		return CONTINUAR_EJECUCION, nil
	case "READ":
		//pasar direccion logica por mmu y mandarle la fisica a memoria y que nos mande lo que lee
		pcb.PC++
		return CONTINUAR_EJECUCION, nil
	case "GOTO":
		nuevoPC, err := strconv.ParseInt(instDeco.Parametros[0], 10, 64)
		if err != nil {
			fmt.Printf("Error al convertir '%s' a int64: %s\n", instDeco.Parametros[0], err)
		}
		pcb.PC = nuevoPC
		return CONTINUAR_EJECUCION, nil
	case "IO":
		//aca va la Logica
		return PONERSE_ESPERA, nil
	case "INIT_PROC":
		//aca va la Logica
		return PONERSE_ESPERA, nil
	case "DUMP_MEMORY":
		//aca va la Logica
		return PONERSE_ESPERA, nil
	case "EXIT":
		//aca va la Logica
		return PONERSE_ESPERA, nil
	}
	return PONERSE_ESPERA, fmt.Errorf("instruccion desconocida: %s", instDeco.Nombre)
}

func RecibirProcesoAEjecutar(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var proc globals.ProcesoAExecutar
	err := decoder.Decode(&proc)
	if err != nil {
		log.Printf("Error al decodificar mensaje: %s\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar mensaje"))
		return
	}

	log.Println("Me llego un proceso paaaa")
	log.Printf("%+v\n", proc)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}
