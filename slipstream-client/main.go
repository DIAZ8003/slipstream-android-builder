package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
)

type Config struct {
	Resolver        string
	Domain          string
	KeepAlive       time.Duration
	TcpListenPort   int
	Congestion      string
	GSO             bool
}

func main() {
	cfg := loadFlags()

	fmt.Println("Slipstream Client (Go)")
	fmt.Println("Resolver:", cfg.Resolver)
	fmt.Println("Domain:", cfg.Domain)
	fmt.Println("TCP Port:", cfg.TcpListenPort)

	// Resolver DNS -> servidor QUIC real
	serverAddr := resolveDomain(cfg.Resolver, cfg.Domain)

	// Listener local TCP (ej: 5201 o 2222)
	local := fmt.Sprintf("127.0.0.1:%d", cfg.TcpListenPort)
	tcpListener, err := net.Listen("tcp", local)
	if err != nil {
		log.Fatalf("Error abriendo el puerto local %s: %v", local, err)
	}
	fmt.Println("Escuchando TCP local en", local)

	// LOOP PRINCIPAL
	for {
		conn, err := tcpListener.Accept()
		if err != nil {
			fmt.Println("Error aceptando conexión:", err)
			continue
		}

		go handleConnection(conn, serverAddr, cfg)
	}
}

func loadFlags() Config {
	resolver := flag.String("resolver", "8.8.8.8:53", "Servidor DNS")
	domain := flag.String("domain", "", "Dominio NS del túnel")
	tcpPort := flag.Int("tcp-listen-port", 5201, "Puerto TCP local")
	keepAlive := flag.Int("keep-alive-interval", 120000, "KeepAlive en ms")
	cc := flag.String("congestion-control", "cubic", "Algoritmo Congestion Control")
	gso := flag.Bool("gso", false, "Enable GSO")

	flag.Parse()

	return Config{
		Resolver:      *resolver,
		Domain:        *domain,
		TcpListenPort: *tcpPort,
		KeepAlive:     time.Duration(*keepAlive) * time.Millisecond,
		Congestion:    *cc,
		GSO:           *gso,
	}
}

func resolveDomain(resolver, domain string) string {
	r := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{}
			return d.DialContext(ctx, "udp", resolver)
		},
	}

	ips, err := r.LookupHost(context.Background(), domain)
	if err != nil || len(ips) == 0 {
		fmt.Println("Error resolviendo dominio:", err)
		os.Exit(1)
	}

	fmt.Println("Dirección real del servidor:", ips[0])
	return ips[0] + ":5300"
}

func handleConnection(localConn net.Conn, serverAddr string, cfg Config) {
	defer localConn.Close()

	session, err := quic.DialAddr(
		context.Background(),
		serverAddr,
		&tls.Config{InsecureSkipVerify: true, NextProtos: []string{"h3"}},
		&quic.Config{},
	)
	if err != nil {
		fmt.Println("Error conectando QUIC:", err)
		return
	}
	defer session.CloseWithError(0, "bye")

	stream, err := session.OpenStreamSync(context.Background())
	if err != nil {
		fmt.Println("Error abriendo stream QUIC:", err)
		return
	}
	defer stream.Close()

	// Copia bidireccional TCP <-> QUIC
	go io.Copy(stream, localConn)
	io.Copy(localConn, stream)
}
