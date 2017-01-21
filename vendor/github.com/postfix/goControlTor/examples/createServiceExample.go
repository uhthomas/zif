package main

import (
	"fmt"
	"github.com/david415/goControlTor"
)

func main() {
	torControl := &goControlTor.TorControl{}

	// directory and parent directory must be owned by the tor user/group
	// and have g+rx permissions
	secretServiceDir := "/var/lib/tor-alpha-services/hiddenService01"
	secretServicePort := map[int]string{80: "127.0.0.1:80"}

	var err error = nil
	//torControlNetwork := "tcp"
	// your tor control port is usually 9051
	//torControlAddr := "127.0.0.1:9951"
	//err = torControl.Dial(torControlNetwork, torControlAddr)

	err = torControl.Dial("unix", "/var/lib/tor-alpha-control/control")
	fmt.Print("dialed!\n")
	if err != nil {
		fmt.Print("connect fail %s\n", err)
		return
	}

	// set this to your tor control port authentication password
	//torControlAuthPassword := "toositai8uRupohnugiCeekiex5phahx"
	//err = torControl.PasswordAuthenticate(torControlAuthPassword)

	// directory must be owned by the tor user/group
	// and have g+rx permissions
	//err = torControl.SafeCookieAuthenticate("/var/lib/tor-alpha-control/control_auth_cookie")
	err = torControl.CookieAuthenticate("/var/lib/tor-alpha-control/control_auth_cookie")
	if err != nil {
		fmt.Print("Tor control port cookie authentication fail\n")
		return
	}
	fmt.Print("Tor control port password authentication successful.\n")

	err = torControl.CreateHiddenService(secretServiceDir, secretServicePort)
	if err != nil {
		fmt.Printf("create hidden service fail: %s\n", err)
		return
	}
	fmt.Print("Tor hidden service created.\n")

	// XXX
	onion := ""
	onion, err = goControlTor.ReadOnion(secretServiceDir)
	if err != nil {
		fmt.Printf("ReadOnion error: %s\n", err)
		return
	}
	fmt.Printf("hidden service onion: %s\n", onion)

}
