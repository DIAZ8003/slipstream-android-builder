package main

import (
    "flag"
    "fmt"
)

func main() {
    server := flag.String("server", "127.0.0.1:5300", "Dirección del servidor Slipstream")
    domain := flag.String("domain", "ns.etecsa.news", "Dominio usado por Slipstream")
    flag.Parse()

    fmt.Println("Cliente Slipstream (dummy) iniciado")
    fmt.Println("Servidor:", *server)
    fmt.Println("Dominio :", *domain)
}
