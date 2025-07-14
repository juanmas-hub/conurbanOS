package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io" //NUEVO Necesario para io.ReadAll
	"log"
	"log/slog"
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

func IniciarConfiguracionMemoria(filePath string) *globals.Memoria_Config {
	var config *globals.Memoria_Config
	configFile, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer configFile.Close()

	jsonParser := json.NewDecoder(configFile)
	jsonParser.Decode(&config)

	return config
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

var Tamañopag = 0

func Execute(instDeco globals.InstruccionDecodificada, pcb *globals.PCB) (ResultadoEjecucion, error) {
	switch instDeco.Nombre { //En cada caso habria que extraer los parametros del string y pasarlos a una variable de su tipo de dato, luego ejecutar la logica correspondiente
	// Tambien hay que actualizar el PC, hacerle ++ o actualizarlo al valor del GOTO
	case "NOOP":
		time.Sleep(1 * time.Second)
		pcb.PC++
		return CONTINUAR_EJECUCION, nil
	case "WRITE":
		direccionLog, err := strconv.ParseInt(instDeco.Parametros[0], 10, 64)
		if err != nil {
			fmt.Printf("Error al convertir '%s': %v\n", instDeco.Parametros[0], err)
		} else {
			fmt.Printf("'%s' convertido a int64: %d\n", instDeco.Parametros[0], direccionLog)
		}

		fmt.Println("DEBUG: CpuConfig es nil?", globals.CpuConfig == nil)
		fmt.Println("DEBUG: MemoriaConfig es nil?", globals_cpu.MemoriaConfig == nil)

		fmt.Printf("DEBUG: Page_size: %d\n", globals_cpu.MemoriaConfig.Page_size)
		fmt.Printf("DEBUG: Tlb_entries: %d\n", globals.CpuConfig.Tlb_entries)
		fmt.Printf("DEBUG: Number_of_levels: %d\n", globals_cpu.MemoriaConfig.Number_of_levels)
		fmt.Printf("DEBUG: Entries_per_page: %d\n", globals_cpu.MemoriaConfig.Entries_per_page)
		fmt.Printf("DEBUG: Cache_entries: %d\n", globals_cpu.CpuConfig.Cache_entries)

		entradas, desplazamiento, paginaVirtual := ExtraerEntradasYDesplazamiento(direccionLog, globals_cpu.MemoriaConfig.Page_size, globals_cpu.MemoriaConfig.Entries_per_page, globals_cpu.MemoriaConfig.Number_of_levels)

		if globals.CpuConfig.Cache_entries > 0 { //si hay cache

			entradaCache, encontrado, err := BuscarPaginaEnCache(paginaVirtual, pcb.Pid)
			if err != nil {
				fmt.Printf("Error al buscar pagina en cache: %s\n", err)
			}

			if !encontrado {
				fmt.Printf("Cache MISS\n")
				direccionFisica, err := ConseguirDireccionFisica(paginaVirtual, desplazamiento, pcb.Pid, entradas)
				if err != nil {
					fmt.Printf("Error al conseguir direccion fisica: %s\n", err)
				}
				contenidoPag, err := PedirContenidoPagina(direccionFisica)
				if err != nil {
					fmt.Printf("Error al pedir contenido de pagina: %s\n", err)
				}
				entradaCache = &globals.CacheEntry{
					Pagina:    paginaVirtual,
					Contenido: contenidoPag,
					PID:       pcb.Pid,
					R:         true,
					D:         true}
				InsertarOReemplazarEnCache(entradaCache)
			} else {
				fmt.Printf("entradaCache encontrada para pagina: %d\n", entradaCache.Pagina)

			}
			err = EscribirCache(entradaCache, desplazamiento, instDeco.Parametros[1])
			if err != nil {
				fmt.Printf("Error al escribir en cache")
			} else {
				fmt.Printf("Se escribio en cache correctamente")
			}

			slog.Debug(fmt.Sprint("Se actualizo la cache: ", globals.ElCache))
		} else { //si no hay cache
			direcccionFisica, err := ConseguirDireccionFisica(paginaVirtual, desplazamiento, pcb.Pid, entradas)
			slog.Debug(fmt.Sprint("Direccion fisica: ", direcccionFisica))
			if err != nil {
				fmt.Printf("Error al conseguir direccion fisica: %s\n", err)
			}
			EscribirDatoMemoria(direcccionFisica, instDeco.Parametros[1], pcb.Pid)
		}
		pcb.PC++
		return CONTINUAR_EJECUCION, nil
	case "READ":
		//pasar direccion logica por mmu y mandarle la fisica a memoria y que nos mande lo que lee
		direccionLog, err := strconv.ParseInt(instDeco.Parametros[0], 10, 64)
		if err != nil {
			fmt.Printf("Error al convertir '%s': %v\n", instDeco.Parametros[0], err)
		} else {
			fmt.Printf("'%s' convertido a int64: %d\n", instDeco.Parametros[0], direccionLog)
		}
		tamanio, err := strconv.ParseInt(instDeco.Parametros[1], 10, 64)
		if err != nil {
			fmt.Printf("Error al convertir '%s': %v\n", instDeco.Parametros[0], err)
		} else {
			fmt.Printf("'%s' convertido a int64: %d\n", instDeco.Parametros[0], direccionLog)
		}

		fmt.Println("DEBUG: CpuConfig es nil?", globals.CpuConfig == nil)
		fmt.Println("DEBUG: MemoriaConfig es nil?", globals_cpu.MemoriaConfig == nil)

		fmt.Printf("DEBUG: Page_size: %d\n", globals_cpu.MemoriaConfig.Page_size)
		fmt.Printf("DEBUG: Tlb_entries: %d\n", globals.CpuConfig.Tlb_entries)
		fmt.Printf("DEBUG: Number_of_levels: %d\n", globals_cpu.MemoriaConfig.Number_of_levels)
		fmt.Printf("DEBUG: Entries_per_page: %d\n", globals_cpu.MemoriaConfig.Entries_per_page)
		fmt.Printf("DEBUG: Cache_entries: %d\n", globals_cpu.CpuConfig.Cache_entries)

		entradas, desplazamiento, paginaVirtual := ExtraerEntradasYDesplazamiento(direccionLog, globals_cpu.MemoriaConfig.Page_size, globals_cpu.MemoriaConfig.Entries_per_page, globals_cpu.MemoriaConfig.Number_of_levels)

		if globals.CpuConfig.Cache_entries > 0 { //si hay cache

			entradaCache, encontrado, err := BuscarPaginaEnCache(paginaVirtual, pcb.Pid)
			if err != nil {
				fmt.Printf("Error al buscar pagina en cache: %s\n", err)
			}

			if !encontrado {
				fmt.Printf("Cache MISS\n")
				direccionFisica, err := ConseguirDireccionFisica(paginaVirtual, desplazamiento, pcb.Pid, entradas)
				if err != nil {
					fmt.Printf("Error al conseguir direccion fisica: %s\n", err)
				}
				contenidoPag, err := PedirContenidoPagina(direccionFisica)
				if err != nil {
					fmt.Printf("Error al pedir contenido de la pagina: %s\n", err)
				}
				entradaCache = &globals.CacheEntry{
					Pagina:    paginaVirtual,
					Contenido: contenidoPag,
					PID:       pcb.Pid,
					R:         true,
					D:         false}
				InsertarOReemplazarEnCache(entradaCache)
				slog.Debug(fmt.Sprint("Se actualizo la cache: ", globals.ElCache))
				//slog.Debug("Se actualizo la cache: ", globals.ElCache)
			} else {
				fmt.Printf("entradaCache encontrada para pagina: %d\n", entradaCache.Pagina)
			}
			contenidoLeido, err := LeerDeCache(entradaCache, desplazamiento, tamanio)
			if err != nil {
				fmt.Printf("Error al leer en cache")
			} else {
				fmt.Printf("leyo en cache correctamente")
				fmt.Printf("Lectura: %s\n", contenidoLeido)
			} //FALTA MOSTRAR DATO LEIDO POR LOG Y CONSOLA

		} else { //si no hay cache
			direcccionFisica, err := ConseguirDireccionFisica(paginaVirtual, desplazamiento, pcb.Pid, entradas)
			if err != nil {
				fmt.Printf("Error al conseguir direccion fisica")
			}
			LeerPaginaMemoria(direcccionFisica, tamanio, pcb.Pid)
		}
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

func ConseguirDireccionFisica(paginaVirtual int64, desplazamiento int64, pid int64, entradas []int64) (int64, error) {
	if globals.CpuConfig.Tlb_entries > 0 {
		marco, encontrado := BuscarMarcoEnTLB(paginaVirtual, pid)
		if encontrado {
			direccionFisica := TraducirLogicaAFisica(marco, desplazamiento, globals_cpu.MemoriaConfig.Page_size)
			return direccionFisica, nil
		} else { //si hay tlb pero no esta el marco pide marco a memoria
			marco, err := PedirMarcoDePagina(pid, entradas)
			if err != nil {
				return -1, fmt.Errorf("Error pidiendo marco")
			}
			direccionFisica := TraducirLogicaAFisica(marco, desplazamiento, globals_cpu.MemoriaConfig.Page_size)
			CargarTLB(paginaVirtual, marco, pid, globals.Tlb)
			return direccionFisica, nil
		}
	} else { //si no hay tlb pide marco a memoria
		marco, err := PedirMarcoDePagina(pid, entradas)
		if err != nil {
			return -1, fmt.Errorf("Error pidiendo marco")
		}
		direccionFisica := TraducirLogicaAFisica(marco, desplazamiento, globals_cpu.MemoriaConfig.Page_size)
		return direccionFisica, nil
	}

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

	url := fmt.Sprintf("http://%s:%d/syscallDUMP_MEMORY", globals.CpuConfig.Ip_kernel, globals.CpuConfig.Port_kernel)

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

	log.Printf("SYSCALL enviada correctamente a Kernel (%s). Respuesta: %s", "DUMP", resp.Status)
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

func NuevaCache(capacidad int64, algoritmo string) { //INICIALIZA EL CACHE
	globals.ElCache = &globals.Cache{
		Entries:            make([]*globals.CacheEntry, 0, capacidad), //crea lista de paginas con capacidad de paginas de cache definida (capacidad viene en config)
		PaginaIndex:        make(map[int64]int),                       //genera el map
		Capacidad:          capacidad,
		AlgoritmoReemplazo: algoritmo,
		ClockHand:          0,
	}
}

func NuevaTLB(capacidad int64, algoritmo string) { //CREA LA TLB
	globals_cpu.Tlb = &globals.TLB{
		Entries:            make([]globals.TLBentry, 0, capacidad), //crea lista de paginas con capacidad de la TLB (capacidad viene en config)
		PaginaIndex:        make(map[int64]int),
		Capacidad:          capacidad,
		AlgoritmoReemplazo: algoritmo,
		FIFOindex:          0,
	}
}

//FUNCIONES QUE NOS FALTAN

func BuscarPaginaEnCache(paginaVirtual int64, pid int64) (*globals.CacheEntry, bool, error) {

	// 1. Intentar encontrar la página virtual en el mapa auxiliar
	index, found := globals.ElCache.PaginaIndex[paginaVirtual]

	if found {
		// ¡Cache Hit! La página fue encontrada.
		// AHORA, ¡VERIFICAR EL PID! Esto es CRÍTICO para el aislamiento entre procesos.
		// Una página virtual con el mismo número pero de un PID diferente NO es un hit.
		if globals.ElCache.Entries[index].PID != pid {
			log.Printf("PID %d: Cache Miss por PID mismatch para Pagina %d (Encontró PID %d en caché)",
				pid, paginaVirtual, globals.ElCache.Entries[index].PID)
			return nil, false, nil // Es un miss para este PID
		}

		// Si llegamos aquí, la página y el PID coinciden: ¡Es un verdadero Cache Hit!
		entradaCache := globals.ElCache.Entries[index] // Obtenemos una referencia a la entrada

		// 2. Actualizar el bit de referencia para algoritmos de reemplazo (CLOCK/CLOCK-M)
		entradaCache.R = true // Se ha accedido a esta página

		log.Printf("PID %d: Cache Hit para Pagina %d(Referenced bit actualizado)",
			pid, paginaVirtual) // Asumiendo que Marco es parte de PageCacheEntry

		return entradaCache, true, nil // Devolvemos la entrada y true (indicando hit)

	} else {
		// Cache Miss: La página no fue encontrada en la caché.
		log.Printf("PID %d: Cache Miss para Pagina %d", pid, paginaVirtual)
		return nil, false, nil // Devolvemos nil, false (indicando miss) y sin error
	}
}

func BuscarMarcoEnTLB(paginaVirtual int64, pid int64) (int64, bool) {
	index, found := globals.Tlb.PaginaIndex[paginaVirtual]
	if found {

		if globals.Tlb.Entries[index].PID != pid {
			log.Printf("PID %d: TLB Miss por PID mismatch para Pagina %d (Encontró PID %d)",
				pid, paginaVirtual, globals.Tlb.Entries[index].PID)
			return -1, false // No es un hit para este PID
		}

		marco := globals.Tlb.Entries[index].Marco

		// Si el algoritmo es LRU, actualizamos el Timestamp para marcarla como "usada recientemente"
		if globals.Tlb.AlgoritmoReemplazo == "LRU" {
			globals.Tlb.Entries[index].Timestamp = time.Now().UnixNano()
			log.Printf("PID %d: TLB Hit para Pagina %d, Marco %d (LRU: Timestamp actualizado)", paginaVirtual, marco, pid)
		} else {
			log.Printf("PID %d: TLB Hit para Pagina %d, Marco %d", pid, paginaVirtual, marco)
		}

		return marco, true // Devolvemos el marco encontrado y true (indicando hit)

	} else {
		// TLB Miss: La traducción no fue encontrada en la TLB.
		log.Printf("PID %d: TLB Miss para Pagina %d", pid, paginaVirtual)
		return -1, false // Devolvemos -1 y false (indicando miss)
	}
}

// (1) funcion que extraiga Entrada, Indice y Desplazamiento de la direccion logica
// Devuelve:
// entradas: slice con la entrada correspondiente a cada nivel (de 1 a N),
// desplazamiento: el desplazamiento dentro de la página.
func ExtraerEntradasYDesplazamiento(direccionLogica, tamanioPagina, cantEntradasTabla, niveles int64) ([]int64, int64, int64) {
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
	return entradas, desplazamiento, nroPagina
}

// (2) funcion que traduzca funcion logica a fisica teniendo el marco
func TraducirLogicaAFisica(marco int64, desplazamiento int64, tamanioPagina int64) int64 {
	return marco*tamanioPagina + desplazamiento
}

// (3) funcion que inserte o reemplace en CACHE cuando esta lleno, con el algoritmo elegido
func InsertarOReemplazarEnCache(nueva *globals.CacheEntry) {
	// Si la página ya está en caché, la actualiza y setea Referenced
	if idx, ok := globals.ElCache.PaginaIndex[nueva.Pagina]; ok {
		*globals.ElCache.Entries[idx] = *nueva
		globals.ElCache.Entries[idx].R = true
		slog.Debug(fmt.Sprint("La pagina esta en cache"))
		return
	}

	// Si hay espacio, inserta la nueva página
	if int64(len(globals.ElCache.Entries)) < globals.ElCache.Capacidad {
		globals.ElCache.Entries = append(globals.ElCache.Entries, nueva)
		globals.ElCache.PaginaIndex[nueva.Pagina] = len(globals.ElCache.Entries) - 1
		slog.Debug(fmt.Sprint("Hay espacio en cache, inserto entrada"))
		return
	}

	// Cache llena: elegir víctima según algoritmo
	var victimaIdx int
	switch globals.ElCache.AlgoritmoReemplazo {
	case "CLOCK":
		victimaIdx = buscarVictimaCLOCK()
	case "CLOCK-M":
		victimaIdx = buscarVictimaCLOCKM()
	default:
		panic("Algoritmo de reemplazo no soportado")
	}

	// Si la víctima está modificada, escribir su contenido a memoria principal
	if globals.ElCache.Entries[victimaIdx].D {
		victima := globals.ElCache.Entries[victimaIdx]
		pid := victima.PID
		contenido := victima.Contenido

		direccionLogica := victima.Pagina * globals.MemoriaConfig.Page_size
		entradas, _, _ := ExtraerEntradasYDesplazamiento(direccionLogica, globals.MemoriaConfig.Page_size, globals.MemoriaConfig.Entries_per_page, globals.MemoriaConfig.Number_of_levels)
		direccionFisica, errorr := ConseguirDireccionFisica(victima.Pagina, 0, pid, entradas)
		if errorr != nil {
			log.Printf("⚠️ Error al conseguir la direccion fisica de la victima en InsertarOReemplazarEnCache")
		}
		err := ActualizarPaginaMemoria(pid, direccionFisica, contenido)
		if err != nil {
			log.Printf("⚠️ Error al actualizar página modificada de PID %d, direccion %d: %v", pid, direccionFisica, err)
		}
		slog.Debug(fmt.Sprint("Se actualizo la pagina en memoria PID: ", pid))
	}

	// Eliminar la página víctima del índice
	delete(globals.ElCache.PaginaIndex, globals.ElCache.Entries[victimaIdx].Pagina)

	// Reemplazar entrada
	globals.ElCache.Entries[victimaIdx] = nueva
	globals.ElCache.PaginaIndex[nueva.Pagina] = victimaIdx

	// Avanzar el clock hand
	globals.ElCache.ClockHand = (victimaIdx + 1) % len(globals.ElCache.Entries)

	slog.Debug(fmt.Sprint("Se reemplazo una pagina en cache"))
}

// buscarVictimaCLOCK devuelve el índice de la víctima según el algoritmo CLOCK.
func buscarVictimaCLOCK() int {
	for {
		if !globals.ElCache.Entries[globals.ElCache.ClockHand].R {
			return globals.ElCache.ClockHand
		}
		globals.ElCache.Entries[globals.ElCache.ClockHand].R = false
		globals.ElCache.ClockHand = (globals.ElCache.ClockHand + 1) % len(globals.ElCache.Entries)
	}
}

// buscarVictimaCLOCKM devuelve el índice de la víctima según el algoritmo CLOCK-M.
func buscarVictimaCLOCKM() int {
	// Primera pasada: R=0 y D=0
	for i := 0; i < len(globals.ElCache.Entries); i++ {
		idx := (globals.ElCache.ClockHand + i) % len(globals.ElCache.Entries)
		if !globals.ElCache.Entries[idx].R && !globals.ElCache.Entries[idx].D {
			globals.ElCache.ClockHand = (idx + 1) % len(globals.ElCache.Entries)
			return idx
		}
	}
	// Segunda pasada: R=0 y D=1 (y setea R=0 para futuras vueltas)
	for i := 0; i < len(globals.ElCache.Entries); i++ {
		idx := (globals.ElCache.ClockHand + i) % len(globals.ElCache.Entries)
		if !globals.ElCache.Entries[idx].R && globals.ElCache.Entries[idx].D {
			globals.ElCache.ClockHand = (idx + 1) % len(globals.ElCache.Entries)
			return idx
		}
		globals.ElCache.Entries[idx].R = false
	}
	// Si todos tenían R=1, vuelve a intentar
	return buscarVictimaCLOCKM()
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
func PedirMarcoDePagina(pid int64, entradas []int64) (int64, error) {
	solicitud := struct {
		Pid      int64   `json:"pid"`
		Entradas []int64 `json:"entradas"`
	}{
		Pid:      pid,
		Entradas: entradas,
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
func PedirContenidoPagina(direccionFisica int64) ([]byte, error) {

	if direccionFisica%globals.MemoriaConfig.Page_size != 0 {
		direccionFisica = direccionFisica - direccionFisica%globals.MemoriaConfig.Page_size
	}

	mensaje := globals_cpu.LeerPaginaDTO{
		DireccionFisica: direccionFisica,
	}

	slog.Debug(fmt.Sprint("Direccion fisica enviada a memoria en PedirContenidoPagina: ", direccionFisica))

	body, err := json.Marshal(mensaje)
	if err != nil {
		return []byte{}, fmt.Errorf("error codificando solicitud: %w", err)
	}

	url := fmt.Sprintf("http://%s:%d/leerPagina", globals_cpu.CpuConfig.Ip_memory, globals_cpu.CpuConfig.Port_memory)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return []byte{}, fmt.Errorf("error haciendo POST a Memoria: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return []byte{}, fmt.Errorf("memoria respondió con error: %d", resp.StatusCode)
	}

	var respuesta globals_cpu.RespuestaContenido
	err = json.NewDecoder(resp.Body).Decode(&respuesta)
	if err != nil {
		return []byte{}, fmt.Errorf("error al decodificar respuesta: %w", err)
	}
	slog.Debug(fmt.Sprint("Contenido recibido en PedirContenidoPagina: ", respuesta.Contenido))

	return respuesta.Contenido, nil
}

func EscribirDatoMemoria(direccionFisica int64, dato string, pid int64) error {
	solicitudEscritura := globals.SolicitudEscritura{Pid: pid, DireccionFisica: direccionFisica, Dato: dato}

	slog.Debug(fmt.Sprint("Solicitud escritura: ", solicitudEscritura))

	body, err := json.Marshal(solicitudEscritura)
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
func LeerPaginaMemoria(direccionFisica int64, tamanio int64, pid int64) ([]byte, error) {
	solicitudLectura := globals_cpu.SolicitudLectura{Pid: pid, Posicion: direccionFisica, Tamanio: tamanio}

	body, err := json.Marshal(solicitudLectura)
	if err != nil {
		return []byte{}, fmt.Errorf("error codificando solicitud: %w", err)
	}

	url := fmt.Sprintf("http://%s:%d/accederEspacioUsuarioLectura", globals_cpu.CpuConfig.Ip_memory, globals_cpu.CpuConfig.Port_memory)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return []byte{}, fmt.Errorf("error haciendo POST a Memoria: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return []byte{}, fmt.Errorf("memoria respondió con error: %d", resp.StatusCode)
	}

	var respuesta globals_cpu.RespuestaContenido
	err = json.NewDecoder(resp.Body).Decode(&respuesta)
	if err != nil {
		return []byte{}, fmt.Errorf("error al decodificar respuesta: %w", err)
	}

	return respuesta.Contenido, nil
}

// (9) funcion que pida a memoria actualizar una pagina (Dirty BIT),
func ActualizarPaginaMemoria(pid int64, direccionFisica int64, contenido []byte) error {
	solicitud := globals_cpu.SolicitudPaginaContenido{Pid: pid, DireccionFisica: direccionFisica, Contenido: contenido}

	slog.Debug(fmt.Sprint("Enviado a memoria para actualizar pagina: ", solicitud))

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
func EscribirCache(entradaCache *globals.CacheEntry, desplazamiento int64, dato string) error {
	bytesAEscribir := []byte(dato)
	tamanio := int64(len(bytesAEscribir))
	if desplazamiento+tamanio > 64 {
		return fmt.Errorf("intento de lectura excede los límites de la página en caché")
	}
	copy(entradaCache.Contenido[desplazamiento:], bytesAEscribir)
	entradaCache.R = true
	entradaCache.D = true

	fmt.Printf("DEBUG: dirección entradaCache: %p\n", entradaCache)

	return nil
}

// (11) funcion que lea en cache
func LeerDeCache(entradaCache *globals.CacheEntry, desplazamiento int64, tamanio int64) ([]byte, error) {
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

func EnviarPCBaKernel(pid int64, pc int64) error {
	pcYpid := globals_cpu.PCyPID{Pid: pid, Pc: pc}

	body, err := json.Marshal(pcYpid)
	if err != nil {
		return fmt.Errorf("error codificando solicitud: %w", err)
	}

	url := fmt.Sprintf("http://%s:%d/PCyPIDdevueltos", globals_cpu.CpuConfig.Ip_memory, globals_cpu.CpuConfig.Port_memory)
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

// TEMPORAL -- para probar
func Wait(semaforo globals.Semaforo) {
	<-semaforo
}

func Signal(semaforo globals.Semaforo) {
	semaforo <- struct{}{}
}

// Interrupcion
func RecibirInterrupcion(w http.ResponseWriter, r *http.Request) {
	var interrupt globals.Interrupcion
	err := json.NewDecoder(r.Body).Decode(&interrupt)
	if err != nil {
		log.Printf("Error al decodificar PID: %s\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar PID"))
		return
	}

	log.Printf("PID recibido para interrumpir: %d\n", interrupt.PID)

	globals.HayInterrupcion = true

	log.Print("Esperando que termine el ciclo de instruccion")
	Wait(globals.Sem_Interrupcion)
	log.Print("Ya se interrumpio")

	respuesta := globals.RespuestaInterrupcion{
		PC: globals.PC_Interrupcion,
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(respuesta)
	if err != nil {
		log.Printf("Error al codificar respuesta: %s\n", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
