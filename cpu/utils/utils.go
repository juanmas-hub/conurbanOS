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
		log.Println("Parametros antes de dividir: ", parametrosStr)
	}

	parametros := strings.Split(parametrosStr, " ") //divide los argumentos y los deja separados en un array de strings
	log.Println("Parametros despues de dividir: ", parametros)
	if parametrosStr == "" {
		parametros = []string{}
	}

	log.Println("longitud: ", len(parametros))

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
		// Deberia ser != 0, pero lo cambie porque el len parametros anda mal.
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

func Execute(instDeco globals.InstruccionDecodificada, pcb *globals.PCB) (ResultadoEjecucion, error) {
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
		TiempoINT, err := strconv.ParseInt(instDeco.Parametros[1], 10, 64)
		if err != nil {
			fmt.Printf("Error al convertir '%s' a int64: %s\n", instDeco.Parametros[1], err)
		}
		pcb.PC++
		IO := globals.SyscallIO{
			NombreIO:  instDeco.Parametros[0],
			NombreCPU: os.Args[1],
			Tiempo:    TiempoINT,
			PID:       pcb.Pid,
			PC:        pcb.PC}
		err = EnviarIOAKernel(IO)
		if err != nil {
			log.Printf("ERROR: No se pudo enviar SYSCALL IO del PID %d al Kernel: %s", pcb.Pid, err)
			return ERROR_EJECUCION, fmt.Errorf("fallo al enviar SYSCALL IO: %w", err)
		}
		return PONERSE_ESPERA, nil
	case "INIT_PROC":
		TamanioINT, err := strconv.ParseInt(instDeco.Parametros[1], 10, 64)
		if err != nil {
			fmt.Printf("Error al convertir '%s' a int64: %s\n", instDeco.Parametros[1], err)
		}
		pcb.PC++
		INIT := globals.SyscallInit{
			Tamanio:   TamanioINT,
			Archivo:   instDeco.Parametros[0],
			NombreCPU: os.Args[1],
			PID:       pcb.Pid,
			PC:        pcb.PC}
		err = EnviarINITAKernel(INIT)
		if err != nil {
			log.Printf("ERROR: No se pudo enviar SYSCALL INIT del PID %d al Kernel: %s", pcb.Pid, err)
			return ERROR_EJECUCION, fmt.Errorf("fallo al enviar SYSCALL INIT: %w", err)
		}
		return PONERSE_ESPERA, nil
	case "DUMP_MEMORY":
		pcb.PC++
		DUMP := globals.SyscallDump{
			PID:       pcb.Pid,
			PC:        pcb.PC,
			NombreCPU: os.Args[1]}
		err := EnviarDUMPAKernel(DUMP)
		if err != nil {
			log.Printf("ERROR: No se pudo enviar SYSCALL DUMP MEMORY del PID %d al Kernel: %s", pcb.Pid, err)
			return ERROR_EJECUCION, fmt.Errorf("fallo al enviar SYSCALL DUMP MEMORY: %w", err)
		}
		return PONERSE_ESPERA, nil
	case "EXIT":
		EXIT := globals.SyscallExit{
			PID:       pcb.Pid,
			NombreCPU: os.Args[1]}
		err := EnviarEXITAKernel(EXIT)
		if err != nil {
			log.Printf("ERROR: No se pudo enviar SYSCALL EXIT del PID %d al Kernel: %s", pcb.Pid, err)
			return ERROR_EJECUCION, fmt.Errorf("fallo al enviar SYSCALL EXIT: %w", err)
		}
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

	// TEMPORAL
	go func() {
		pcb := globals.PCB{
			Pid: proc.PID,
			PC:  proc.PC,
		}
		ColaDeEjecucion <- pcb
		Signal(globals.Sem)
	}()

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func EnviarIOAKernel(syscallData globals_cpu.SyscallIO) error {
	body, err := json.Marshal(syscallData)
	if err != nil {
		return fmt.Errorf("error codificando struct SYSCALL: %w", err)
	}

	url := fmt.Sprintf("http://%s:%d/syscallIO", globals.CpuConfig.Ip_kernel, globals.CpuConfig.Port_kernel)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("error enviando SYSCALL a Kernel (%s): %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody := new(bytes.Buffer)
		respBody.ReadFrom(resp.Body)
		return fmt.Errorf("kernel respondió con error al recibir SYSCALL (%d %s): %s", resp.StatusCode, resp.Status, respBody.String())
	}

	log.Printf("SYSCALL enviada correctamente a Kernel (%s). Respuesta: %s", "IO", resp.Status)
	return nil
}

func EnviarEXITAKernel(syscallData globals_cpu.SyscallExit) error {
	body, err := json.Marshal(syscallData)
	if err != nil {
		return fmt.Errorf("error codificando struct SYSCALL: %w", err)
	}

	url := fmt.Sprintf("http://%s:%d/syscallEXIT", globals.CpuConfig.Ip_kernel, globals.CpuConfig.Port_kernel)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("error enviando SYSCALL a Kernel (%s): %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody := new(bytes.Buffer)
		respBody.ReadFrom(resp.Body)
		return fmt.Errorf("kernel respondió con error al recibir SYSCALL (%d %s): %s", resp.StatusCode, resp.Status, respBody.String())
	}

	log.Printf("SYSCALL enviada correctamente a Kernel (%s). Respuesta: %s", "EXIT", resp.Status)
	return nil
}

func EnviarDUMPAKernel(syscallData globals_cpu.SyscallDump) error {
	body, err := json.Marshal(syscallData)
	if err != nil {
		return fmt.Errorf("error codificando struct SYSCALL: %w", err)
	}

	url := fmt.Sprintf("http://%s:%d/syscallEXIT", globals.CpuConfig.Ip_kernel, globals.CpuConfig.Port_kernel)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("error enviando SYSCALL a Kernel (%s): %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody := new(bytes.Buffer)
		respBody.ReadFrom(resp.Body)
		return fmt.Errorf("kernel respondió con error al recibir SYSCALL (%d %s): %s", resp.StatusCode, resp.Status, respBody.String())
	}

	log.Printf("SYSCALL enviada correctamente a Kernel (%s). Respuesta: %s", "EXIT", resp.Status)
	return nil
}

func EnviarINITAKernel(syscallData globals_cpu.SyscallInit) error {
	body, err := json.Marshal(syscallData)
	if err != nil {
		return fmt.Errorf("error codificando struct SYSCALL: %w", err)
	}

	url := fmt.Sprintf("http://%s:%d/syscallINIT_PROC", globals.CpuConfig.Ip_kernel, globals.CpuConfig.Port_kernel)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("error enviando SYSCALL a Kernel (%s): %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody := new(bytes.Buffer)
		respBody.ReadFrom(resp.Body)
		return fmt.Errorf("kernel respondió con error al recibir SYSCALL (%d %s): %s", resp.StatusCode, resp.Status, respBody.String())
	}

	log.Printf("SYSCALL enviada correctamente a Kernel (%s). Respuesta: %s", "INIT", resp.Status)
	return nil
}

func NuevaCache(capacidad int64, algoritmo string) *globals.Cache { //CREA EL CACHE
	return &globals.Cache{
		Entries:            make([]globals.CacheEntry, 0, capacidad), //crea lista de paginas con capacidad de paginas de cache definida (capacidad viene en config)
		PaginaIndex:        make(map[int64]int),                      //genera el map
		Capacidad:          capacidad,
		AlgoritmoReemplazo: algoritmo,
		ClockHand:          0,
	}
}

func NuevaTLB(capacidad int64, algoritmo string) *globals.TLB { //CREA LA TLB
	return &globals.TLB{
		Entries:            make([]globals.TLBentry, 0, capacidad), //crea lista de paginas con capacidad de la TLB (capacidad viene en config)
		PaginaIndex:        make(map[int64]int),
		Capacidad:          capacidad,
		AlgoritmoReemplazo: algoritmo,
		FIFOindex:          0,
	}
}

//FUNCIONES QUE NOS FALTAN

// (1) funcion que extraiga Entrada, Indice y Desplazamiento de la direccion logica
// Devuelve:
// entradas: slice con la entrada correspondiente a cada nivel (de 1 a N),
// desplazamiento: el desplazamiento dentro de la página.
func ExtraerEntradasYDesplazamiento(direccionLogica, tamanioPagina, cantEntradasTabla, niveles int64) ([]int64, int64) {
	nroPagina := direccionLogica / tamanioPagina
	entradas := make([]int64, niveles)
	for x := int64(1); x <= niveles; x++ {
		exp := int64(1)
		for i := int64(0); i < (niveles - x); i++ {
			exp *= cantEntradasTabla
		}
		entradaX := (nroPagina / exp) % cantEntradasTabla
		entradas[x-1] = entradaX
	}
	desplazamiento := direccionLogica % tamanioPagina
	return entradas, desplazamiento
}

// (2) funcion que traduzca funcion logica a fisica teniendo el marco
func TraducirLogicaAFisica(marco int64, desplazamiento int64, tamanioPagina int64) int64 {
	return marco*tamanioPagina + desplazamiento
}

// (3) funcion que inserte o reemplace en CACHE cuando esta lleno, con el algoritmo elegido
func InsertarOReemplazarEnCache(c *globals.Cache, nueva globals.CacheEntry, escribirEnMemoria func(entry globals.CacheEntry)) {
	// Si la página ya está en caché, la actualiza y setea Referenced
	if idx, ok := c.PaginaIndex[nueva.Pagina]; ok {
		c.Entries[idx] = nueva
		c.Entries[idx].R = true
		return
	}

	// Si hay espacio, inserta la nueva página
	if int64(len(c.Entries)) < c.Capacidad {
		c.Entries = append(c.Entries, nueva)
		c.PaginaIndex[nueva.Pagina] = len(c.Entries) - 1
		return
	}

	// Cache llena: elegir víctima según algoritmo
	var victimaIdx int
	switch c.AlgoritmoReemplazo {
	case "CLOCK":
		victimaIdx = buscarVictimaCLOCK(c)
	case "CLOCK-M":
		victimaIdx = buscarVictimaCLOCKM(c)
	default:
		panic("Algoritmo de reemplazo no soportado")
	}

	// Si la víctima está modificada, escribir su contenido a memoria principal
	if c.Entries[victimaIdx].D {
		escribirEnMemoria(c.Entries[victimaIdx])
	}

	// Eliminar la página víctima del índice
	delete(c.PaginaIndex, c.Entries[victimaIdx].Pagina)

	// Reemplazar entrada
	c.Entries[victimaIdx] = nueva
	c.PaginaIndex[nueva.Pagina] = victimaIdx

	// Avanzar el clock hand
	c.ClockHand = (victimaIdx + 1) % len(c.Entries)
}

// buscarVictimaCLOCK devuelve el índice de la víctima según el algoritmo CLOCK.
func buscarVictimaCLOCK(c *globals.Cache) int {
	for {
		if !c.Entries[c.ClockHand].R {
			return c.ClockHand
		}
		c.Entries[c.ClockHand].R = false
		c.ClockHand = (c.ClockHand + 1) % len(c.Entries)
	}
}

// buscarVictimaCLOCKM devuelve el índice de la víctima según el algoritmo CLOCK-M.
func buscarVictimaCLOCKM(c *globals.Cache) int {
	// Primera pasada: R=0 y D=0
	for i := 0; i < len(c.Entries); i++ {
		idx := (c.ClockHand + i) % len(c.Entries)
		if !c.Entries[idx].R && !c.Entries[idx].D {
			c.ClockHand = (idx + 1) % len(c.Entries)
			return idx
		}
	}
	// Segunda pasada: R=0 y D=1 (y setea R=0 para futuras vueltas)
	for i := 0; i < len(c.Entries); i++ {
		idx := (c.ClockHand + i) % len(c.Entries)
		if !c.Entries[idx].R && c.Entries[idx].D {
			c.ClockHand = (idx + 1) % len(c.Entries)
			return idx
		}
		c.Entries[idx].R = false
	}
	// Si todos tenían R=1, vuelve a intentar
	return buscarVictimaCLOCKM(c)
}

// (4) funcion que inserte o reemplace en TLB cuando esta llena, con el algoritmo elegido
func ReemplazarEnTLB(nuevaEntrada globals.TLBentry, tlb *globals.TLB) error {

	if len(tlb.Entries) < int(tlb.Capacidad) {
		return fmt.Errorf("error interno: la TLB no está llena, no se debería llamar a ReemplazarEnTLB")
	}

	var indiceVictima int

	switch tlb.AlgoritmoReemplazo {
	case "FIFO":
		indiceVictima = tlb.FIFOindex

	case "LRU":
		minTimestamp := int64(-1)
		foundVictim := false

		for i, entry := range tlb.Entries {
			if !foundVictim || entry.Timestamp < minTimestamp {
				minTimestamp = entry.Timestamp
				indiceVictima = i
				foundVictim = true
			}
		}

		if !foundVictim {
			return fmt.Errorf("error LRU: no se encontró una víctima para reemplazar")
		}

	default:
		return fmt.Errorf("algoritmo de reemplazo TLB '%s' no reconocido", tlb.AlgoritmoReemplazo)
	}

	// Eliminar la entrada de la víctima del map auxiliar
	delete(tlb.PaginaIndex, tlb.Entries[indiceVictima].Pagina)

	// Reemplazar la entrada en el slice
	tlb.Entries[indiceVictima] = nuevaEntrada

	// Actualizar el map auxiliar con la nueva entrada y su índice
	tlb.PaginaIndex[nuevaEntrada.Pagina] = indiceVictima

	// Avanzar el puntero si es FIFO
	if tlb.AlgoritmoReemplazo == "FIFO" {
		tlb.FIFOindex = (tlb.FIFOindex + 1) % int(tlb.Capacidad)
	}

	return nil
}

// (5) funcion que pida a memoria el marco de una pagina
func PedirMarcoDePagina(pid int64, pagina int64) (int64, error) {
	solicitud := struct {
		Pid    int64 `json:"pid"`
		Pagina int64 `json:"pagina"`
	}{
		Pid:    pid,
		Pagina: pagina,
	}

	body, err := json.Marshal(solicitud)
	if err != nil {
		return 0, fmt.Errorf("error codificando solicitud de marco a Memoria: %w", err)
	}

	url := fmt.Sprintf("http://%s:%d/obtenerMarcoProceso", globals_cpu.CpuConfig.Ip_memory, globals_cpu.CpuConfig.Port_memory)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return 0, fmt.Errorf("error haciendo POST a Memoria: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("memoria respondió con error: %d", resp.StatusCode)
	}

	var respuesta struct {
		Marco int64 `json:"marco"`
	}
	err = json.NewDecoder(resp.Body).Decode(&respuesta)
	if err != nil {
		return 0, fmt.Errorf("error al decodificar respuesta de Memoria: %w", err)
	}

	log.Printf("Marco recibido de Memoria: %d", respuesta.Marco)
	return respuesta.Marco, nil
}

// (6) funcion que pida a memoria el contenido de una pagina
func PedirContenidoPagina(pid int64, pagina int64) ([64]byte, error) {
	solicitud := globals_cpu.SolicitudPagina{Pid: pid, Pagina: pagina}

	body, err := json.Marshal(solicitud)
	if err != nil {
		return [64]byte{}, fmt.Errorf("error codificando solicitud: %w", err)
	}

	url := fmt.Sprintf("http://%s:%d/leerPagina", globals_cpu.CpuConfig.Ip_memory, globals_cpu.CpuConfig.Port_memory)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return [64]byte{}, fmt.Errorf("error haciendo POST a Memoria: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return [64]byte{}, fmt.Errorf("memoria respondió con error: %d", resp.StatusCode)
	}

	var respuesta globals_cpu.RespuestaContenido
	err = json.NewDecoder(resp.Body).Decode(&respuesta)
	if err != nil {
		return [64]byte{}, fmt.Errorf("error al decodificar respuesta: %w", err)
	}

	return respuesta.Contenido, nil
}

// (7) funcion que pida a memoria que escriba (cache deshabilitado)
func EscribirPaginaMemoria(pid int64, pagina int64, contenido [64]byte) error {
	solicitud := globals_cpu.SolicitudPaginaContenido{Pid: pid, Pagina: pagina, Contenido: contenido}

	body, err := json.Marshal(solicitud)
	if err != nil {
		return fmt.Errorf("error codificando solicitud: %w", err)
	}

	url := fmt.Sprintf("http://%s:%d/accederEspacioUsuarioEscritura", globals_cpu.CpuConfig.Ip_memory, globals_cpu.CpuConfig.Port_memory)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("error haciendo POST a Memoria: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("memoria respondió con error: %d", resp.StatusCode)
	}

	return nil
}

// (8) funcion que pida a memoria que lea (cache deshabilitado)
func LeerPaginaMemoria(pid int64, pagina int64) ([64]byte, error) {
	solicitud := globals_cpu.SolicitudPagina{Pid: pid, Pagina: pagina}

	body, err := json.Marshal(solicitud)
	if err != nil {
		return [64]byte{}, fmt.Errorf("error codificando solicitud: %w", err)
	}

	url := fmt.Sprintf("http://%s:%d/accederEspacioUsuarioLectura", globals_cpu.CpuConfig.Ip_memory, globals_cpu.CpuConfig.Port_memory)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return [64]byte{}, fmt.Errorf("error haciendo POST a Memoria: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return [64]byte{}, fmt.Errorf("memoria respondió con error: %d", resp.StatusCode)
	}

	var respuesta globals_cpu.RespuestaContenido
	err = json.NewDecoder(resp.Body).Decode(&respuesta)
	if err != nil {
		return [64]byte{}, fmt.Errorf("error al decodificar respuesta: %w", err)
	}

	return respuesta.Contenido, nil
}

// (9) funcion que pida a memoria actualizar una pagina (Dirty BIT),
func ActualizarPaginaMemoria(pid int64, pagina int64, contenido [64]byte) error {
	solicitud := globals_cpu.SolicitudPaginaContenido{Pid: pid, Pagina: pagina, Contenido: contenido}

	body, err := json.Marshal(solicitud)
	if err != nil {
		return fmt.Errorf("error codificando solicitud: %w", err)
	}

	url := fmt.Sprintf("http://%s:%d/actualizarPagina", globals_cpu.CpuConfig.Ip_memory, globals_cpu.CpuConfig.Port_memory)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("error haciendo POST a Memoria: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("memoria respondió con error: %d", resp.StatusCode)
	}

	return nil
}

// (10) funcion que escriba en cache
func EscribirCache(pagina int64, desplazamiento int64, dato string, cache *globals.Cache) error {
	index, found := cache.PaginaIndex[pagina]
	if !found {
		return fmt.Errorf("error interno: la página %d no se encontró en la caché para el desplazamiento %d", pagina, desplazamiento)
	}
	entradaCache := &cache.Entries[index]
	bytesAEscribir := []byte(dato)
	tamanio := int64(len(bytesAEscribir))
	if desplazamiento+tamanio > 64 {
		return fmt.Errorf("intento de lectura excede los límites de la página en caché")
	}
	copy(entradaCache.Contenido[desplazamiento:], bytesAEscribir)
	entradaCache.R = true
	entradaCache.D = true

	return nil

}

// (11) funcion que lea en cache
func LeerDeCache(pagina int64, desplazamiento int64, tamanio int64, cache *globals.Cache) ([]byte, error) {
	index, found := cache.PaginaIndex[pagina]
	if !found {
		return nil, fmt.Errorf("error interno: la página %d no se encontró en la caché para el desplazamiento %d", pagina, desplazamiento)
	}

	entradaCache := &cache.Entries[index]
	if desplazamiento+tamanio > 64 {
		return nil, fmt.Errorf("intento de lectura excede los límites de la página en caché")
	}
	bytesLeidos := entradaCache.Contenido[desplazamiento : desplazamiento+tamanio]
	entradaCache.R = true

	return bytesLeidos, nil

}

// 12 cargar TLB
func CargarTLB(pV int64, marco int64, pid int64, tlb *globals.TLB) error {
	nuevaEntrada := globals.TLBentry{
		Pagina:    pV,
		Marco:     marco,
		PID:       pid,
		Timestamp: time.Now().UnixNano(),
	}
	if _, found := tlb.PaginaIndex[pV]; found {
		log.Printf("ADVERTENCIA TLB: Se intentó cargar página %d (PID %d) que ya está en TLB. Sobrescribiendo.", pV, pid)
	}
	if len(tlb.Entries) < int(tlb.Capacidad) {
		// TLB no está llena: simplemente agrega la nueva entrada al final
		tlb.Entries = append(tlb.Entries, nuevaEntrada)
		// Registra la posición de la nueva entrada en el map auxiliar
		tlb.PaginaIndex[nuevaEntrada.Pagina] = len(tlb.Entries) - 1
		log.Printf("TLB: Entrada agregada (PID %d, Pag %d, Marco %d). TLB Size: %d/%d",
			pid, pV, marco, len(tlb.Entries), tlb.Capacidad)
	} else {
		// TLB está llena: llama a la lógica de reemplazo
		log.Printf("TLB: Llena, aplicando algoritmo %s para reemplazar.", tlb.AlgoritmoReemplazo)
		err := ReemplazarEnTLB(nuevaEntrada, tlb) // Llama a la función de reemplazo que ya hicimos
		if err != nil {
			return fmt.Errorf("fallo al reemplazar en TLB: %w", err)
		}
		log.Printf("TLB: Reemplazo exitoso (PID %d, Pag %d, Marco %d). Algoritmo: %s",
			pid, pV, marco, tlb.AlgoritmoReemplazo)
	}

	return nil

}

// TEMPORAL -- para probar
func Wait(semaforo globals.Semaforo) {
	<-semaforo
}

func Signal(semaforo globals.Semaforo) {
	semaforo <- struct{}{}
}
